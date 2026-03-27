package mock

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	"github.com/google/uuid"
)

// LiveConfig 持续日志生成配置
type LiveConfig struct {
	RPS       int           // 每秒请求数
	FaultRate float64       // 故障概率 (0.0-1.0)
	Duration  time.Duration // 持续时间，0 表示无限
	Scenarios []Scenario    // 可用的故障场景（用于 burst 窗口）
}

// LiveEvent 实时日志事件（用于 SSE 推送）
type LiveEvent struct {
	Timestamp string `json:"timestamp"`
	Namespace string `json:"namespace"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// LiveGenerator 持续日志生成器
type LiveGenerator struct {
	es        *es.Client
	scenarios []Scenario

	mu       sync.Mutex
	running  bool
	cancelFn context.CancelFunc

	// subscribers fan-out 订阅者集合，支持多客户端
	subscribers map[chan LiveEvent]struct{}
}

// NewLiveGenerator 创建持续日志生成器
func NewLiveGenerator(esClient *es.Client, scenarios []Scenario) *LiveGenerator {
	return &LiveGenerator{
		es:          esClient,
		scenarios:   scenarios,
		subscribers: make(map[chan LiveEvent]struct{}),
	}
}

// Subscribe 注册一个事件订阅者，返回接收 channel
func (g *LiveGenerator) Subscribe() chan LiveEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	ch := make(chan LiveEvent, 64)
	g.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe 注销一个事件订阅者
func (g *LiveGenerator) Unsubscribe(ch chan LiveEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.subscribers, ch)
	close(ch)
}

// SetEventHandler 设置实时事件回调（兼容旧接口，CLI 使用）
func (g *LiveGenerator) SetEventHandler(handler func(LiveEvent)) {
	// 此方法保留为向后兼容，但不再是主要事件分发机制
	// 新代码应使用 Subscribe/Unsubscribe
}

// IsRunning 返回生成器是否正在运行
func (g *LiveGenerator) IsRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.running
}

// Start 启动后台持续生成（非阻塞）
func (g *LiveGenerator) Start(cfg LiveConfig) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return fmt.Errorf("live generator already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	g.running = true
	g.cancelFn = cancel
	g.mu.Unlock()

	go func() {
		_ = g.Run(ctx, cfg)
		g.mu.Lock()
		g.running = false
		g.mu.Unlock()
	}()
	return nil
}

// Stop 停止后台生成
func (g *LiveGenerator) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cancelFn != nil {
		g.cancelFn()
		g.cancelFn = nil
	}
}

// Run 阻塞运行持续日志生成
func (g *LiveGenerator) Run(ctx context.Context, cfg LiveConfig) error {
	if cfg.RPS <= 0 {
		cfg.RPS = 5
	}
	if cfg.FaultRate == 0 {
		cfg.FaultRate = 0.1 // 默认 10% 故障率
	}
	if cfg.FaultRate < 0 {
		cfg.FaultRate = 0
	}
	if cfg.FaultRate > 1.0 {
		cfg.FaultRate = 1.0
	}

	prefix := g.es.Prefix()
	interval := time.Second / time.Duration(cfg.RPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Burst 窗口状态
	var burstUntil time.Time
	var burstScenario *Scenario
	// 首次故障注入在 5-10 秒后触发，让演示快速看到效果
	nextBurstCheck := time.Now().Add(randomDuration(5*time.Second, 10*time.Second))

	// 批量缓冲
	batch := make([]map[string]any, 0, 50)
	flushTicker := time.NewTicker(2 * time.Second)
	defer flushTicker.Stop()

	// 超时控制
	var deadline <-chan time.Time
	if cfg.Duration > 0 {
		timer := time.NewTimer(cfg.Duration)
		defer timer.Stop()
		deadline = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			// 刷出残余
			if len(batch) > 0 {
				g.flushBatch(context.Background(), prefix, batch)
			}
			return ctx.Err()
		case <-deadline:
			if len(batch) > 0 {
				g.flushBatch(context.Background(), prefix, batch)
			}
			return nil
		case <-flushTicker.C:
			if len(batch) > 0 {
				g.flushBatch(ctx, prefix, batch)
				batch = batch[:0]
			}
		case now := <-ticker.C:
			// 检查是否进入 burst 窗口
			if now.After(nextBurstCheck) && cfg.FaultRate > 0 {
				// 使用配置的故障率触发 burst 窗口
				scenarios := cfg.Scenarios
				if len(scenarios) == 0 {
					scenarios = g.scenarios // 回退到构造时传入的场景
				}
				if rand.Float64() < cfg.FaultRate && len(scenarios) > 0 {
					s := scenarios[rand.Intn(len(scenarios))]
					burstScenario = &s
					burstDuration := randomDuration(5*time.Second, 10*time.Second)
					burstUntil = now.Add(burstDuration)
				}
				// 下次检查间隔 10-20 秒，比原来 30-60 秒更紧凑
				nextBurstCheck = now.Add(randomDuration(10*time.Second, 20*time.Second))
			}

			// 判断当前是否在 burst 窗口内
			isFault := burstScenario != nil && now.Before(burstUntil)
			if !isFault && burstScenario != nil && now.After(burstUntil) {
				burstScenario = nil // burst 窗口结束
			}

			// 生成一组调用链日志（burst 窗口内传递故障场景，否则传 nil）
			reqUUID := uuid.New().String()
			var activeScenario *Scenario
			if isFault && burstScenario != nil {
				activeScenario = burstScenario
			}
			logs := GenerateCallChain(now, reqUUID, activeScenario)
			batch = append(batch, logs...)

			// 触发事件回调
			g.emitEvents(logs)

			// 缓冲区满时刷出
			if len(batch) >= 50 {
				g.flushBatch(ctx, prefix, batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch 按 namespace 分组批量写入 ES
func (g *LiveGenerator) flushBatch(ctx context.Context, prefix string, batch []map[string]any) {
	date := time.Now().Format("2006.01.02")
	byNS := make(map[string][]map[string]any)
	for _, doc := range batch {
		ns, _ := doc["kubernetes_namespace"].(string)
		byNS[ns] = append(byNS[ns], doc)
	}
	for ns, docs := range byNS {
		index := fmt.Sprintf("%s_%s-%s", prefix, ns, date)
		if err := g.es.BulkIndex(ctx, index, docs); err != nil {
			log.Printf("[live] bulk index %s failed: %v", index, err)
		}
	}
}

// emitEvents 将生成的日志 fan-out 到所有订阅者
func (g *LiveGenerator) emitEvents(logs []map[string]any) {
	g.mu.Lock()
	subs := make([]chan LiveEvent, 0, len(g.subscribers))
	for ch := range g.subscribers {
		subs = append(subs, ch)
	}
	g.mu.Unlock()

	if len(subs) == 0 {
		return
	}
	for _, log := range logs {
		ts, _ := log["@timestamp"].(string)
		ns, _ := log["kubernetes_namespace"].(string)
		msg, _ := log["message"].(string)
		level := extractLevelFromMessage(msg)
		evt := LiveEvent{
			Timestamp: ts,
			Namespace: ns,
			Level:     level,
			Message:   msg,
		}
		for _, ch := range subs {
			select {
			case ch <- evt:
			default: // 丢弃慢消费者
			}
		}
	}
}

// extractLevelFromMessage 从日志 message 中提取级别
func extractLevelFromMessage(msg string) string {
	// 检查文本日志格式: [timestamp] [LEVEL][...]
	if len(msg) > 0 && msg[0] == '[' {
		for _, level := range []string{"ERROR", "WARN", "INFO", "DEBUG"} {
			if containsLevel(msg, level) {
				return level
			}
		}
	}
	// 检查 JSON 格式
	if len(msg) > 0 && msg[0] == '{' {
		for _, level := range []string{"error", "warn", "info", "debug"} {
			if containsLevel(msg, `"level":"`+level+`"`) || containsLevel(msg, `"level": "`+level+`"`) {
				return strings.ToUpper(level)
			}
		}
	}
	return "INFO"
}

// containsLevel 检查字符串是否包含指定关键词
func containsLevel(s, level string) bool {
	return strings.Contains(s, level)
}

// randomDuration 生成 [min, max) 范围的随机时长
func randomDuration(min, max time.Duration) time.Duration {
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
