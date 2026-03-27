// Package handler 提供企业微信回调 HTTP 处理器
package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/infrastructure/wechat"
)

// WechatCallbackHandler 处理企微应用回调请求
// GET  /api/v1/wechat/callback — 企微回调 URL 验证
// POST /api/v1/wechat/callback — 接收消息、路由命令、触发诊断
type WechatCallbackHandler struct {
	app      *wechat.App
	router   *wechat.CommandRouter
	diagCmd  *command.DiagnoseHandler
	bot      *wechat.Bot
}

// NewWechatCallbackHandler 创建企微回调处理器
func NewWechatCallbackHandler(
	app *wechat.App,
	router *wechat.CommandRouter,
	diagCmd *command.DiagnoseHandler,
	bot *wechat.Bot,
) *WechatCallbackHandler {
	return &WechatCallbackHandler{
		app:     app,
		router:  router,
		diagCmd: diagCmd,
		bot:     bot,
	}
}

// ServeHTTP 分发 GET（URL验证）和 POST（消息接收）请求
func (h *WechatCallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerify(w, r)
	case http.MethodPost:
		h.handleMessage(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleVerify 处理企微回调 URL 验证（GET 请求）
// 企微平台在配置回调 URL 时会发送此请求验证服务可达性
func (h *WechatCallbackHandler) handleVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	echostr := q.Get("echostr")

	if echostr == "" {
		http.Error(w, "缺少 echostr 参数", http.StatusBadRequest)
		return
	}

	plaintext, err := h.app.VerifyURL(msgSignature, timestamp, nonce, echostr)
	if err != nil {
		slog.Error("企微 URL 验证失败", "error", err)
		http.Error(w, "验证失败", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, plaintext)
}

// handleMessage 处理企微推送的消息（POST 请求）
// 流程：解密 → 解析 → 路由命令 → 立即回复 → 异步诊断 → 推送结论
func (h *WechatCallbackHandler) handleMessage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")

	// 读取请求体（企微加密 XML）
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 限制 1MB
	if err != nil {
		slog.Error("读取企微消息体失败", "error", err)
		http.Error(w, "读取请求失败", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// 解密消息体
	plainXML, err := h.app.DecryptMessage(msgSignature, timestamp, nonce, body)
	if err != nil {
		slog.Error("企微消息解密失败", "error", err)
		http.Error(w, "消息解密失败", http.StatusBadRequest)
		return
	}

	// 解析文本消息
	msg, err := wechat.ParseTextMessage(plainXML)
	if err != nil {
		// 非文本消息（图片、语音等）暂不处理，返回 200 避免企微重试
		slog.Info("忽略非文本消息", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// 路由命令
	action, input := h.router.Route(msg)

	switch action {
	case "help":
		// 帮助命令：被动回复帮助文本
		h.replyPassive(w, r, timestamp, nonce, msg.FromUserID, wechat.HelpText())

	case "diagnose":
		if input == "" {
			h.replyPassive(w, r, timestamp, nonce, msg.FromUserID, "请提供诊断内容，例如：诊断 prj-ubill 连接池耗尽")
			return
		}

		// 立即回复"正在诊断"，避免超过企微 5 秒被动回复超时
		h.replyPassive(w, r, timestamp, nonce, msg.FromUserID, wechat.FormatPendingReply(input))

		// 异步执行诊断，完成后通过 Bot Webhook 推送结论卡片
		go h.asyncDiagnose(msg.FromUserID, input)

	default:
		w.WriteHeader(http.StatusOK)
	}
}

// replyPassive 发送企微被动回复（XML 格式，需加密）
// 企微要求在 5 秒内返回回复，否则视为超时
func (h *WechatCallbackHandler) replyPassive(
	w http.ResponseWriter,
	r *http.Request,
	timestamp, nonce, toUser, content string,
) {
	// 构造明文回复 XML
	replyXML := fmt.Sprintf(
		`<xml><ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[argus]]></FromUserName><CreateTime>%d</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[%s]]></Content></xml>`,
		toUser,
		time.Now().Unix(),
		content,
	)

	// 加密回复
	encryptedReply, err := h.app.EncryptReply(replyXML, timestamp, nonce)
	if err != nil {
		slog.Error("加密被动回复失败", "error", err)
		// 降级：返回空 200，避免企微重试
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, encryptedReply)
}

// asyncDiagnose 异步执行诊断，完成后通过 Bot 推送 Markdown 卡片
func (h *WechatCallbackHandler) asyncDiagnose(fromUserID, input string) {
	// 使用独立 context，与 HTTP 请求生命周期解耦
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	t, err := h.diagCmd.Handle(ctx, command.DiagnoseCommand{
		Input:  input,
		Source: "wechat",
	})
	if err != nil {
		slog.Error("企微触发诊断失败", "user", fromUserID, "error", err)
		h.pushBotMessage(fmt.Sprintf("诊断触发失败：%v", err))
		return
	}

	// 等待诊断完成（DiagnoseHandler.Handle 内部已异步启动 Agent，这里轮询任务状态）
	// 注：当前实现中 Handle 返回的 Task 在 goroutine 中异步更新，
	// 此处通过简单轮询等待诊断结束，最长等待 10 分钟
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			h.pushBotMessage(fmt.Sprintf("任务 %s 诊断超时，请访问控制台查看详情。", t.ID))
			return
		case <-time.After(3 * time.Second):
		}

		// t 是指针，Agent goroutine 会更新其状态
		switch t.Status {
		case "completed", "recovered":
			card := wechat.FormatDiagnosisCard(t)
			h.pushBotMessage(card)
			return
		case "failed":
			h.pushBotMessage(fmt.Sprintf("任务 %s 诊断失败，请访问控制台查看详情。", t.ID))
			return
		}
	}

	h.pushBotMessage(fmt.Sprintf("任务 %s 诊断超时，请访问控制台查看详情。", t.ID))
}

// pushBotMessage 通过 Bot Webhook 推送 Markdown 消息
func (h *WechatCallbackHandler) pushBotMessage(content string) {
	if h.bot == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.bot.SendMarkdown(ctx, content); err != nil {
		slog.Error("企微 Bot 推送消息失败", "error", err)
	}
}
