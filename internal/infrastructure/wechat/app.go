// Package wechat 提供企业微信应用回调处理
package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// AppConfig 企微应用回调配置
// 从环境变量读取，避免修改 config.go
type AppConfig struct {
	CorpID         string // 企业 ID
	Token          string // 回调 Token（环境变量 WECHAT_TOKEN）
	EncodingAESKey string // 回调消息加解密密钥（环境变量 WECHAT_ENCODING_AES_KEY）
	AgentID        int    // 应用 ID
	Secret         string // 应用 Secret
}

// LoadAppConfigFromEnv 从环境变量加载回调认证配置
// 必填环境变量：WECHAT_TOKEN, WECHAT_ENCODING_AES_KEY
// 其余字段由调用方从 WechatConfig 填入
func LoadAppConfigFromEnv() AppConfig {
	return AppConfig{
		Token:          os.Getenv("WECHAT_TOKEN"),
		EncodingAESKey: os.Getenv("WECHAT_ENCODING_AES_KEY"),
	}
}

// App 企微应用，处理回调 URL 验证与消息加解密
type App struct {
	config AppConfig
	aesKey []byte // 解码后的 32 字节 AES key
}

// NewApp 创建企微应用实例
// EncodingAESKey 为 Base64 编码的 43 字符字符串，解码后得到 32 字节 AES key
func NewApp(cfg AppConfig) (*App, error) {
	if cfg.EncodingAESKey == "" {
		// 未配置则返回空实现，允许非加密场景
		return &App{config: cfg}, nil
	}

	// EncodingAESKey 末尾补 "=" 凑成 44 字符以满足 Base64 标准
	keyBytes, err := base64.StdEncoding.DecodeString(cfg.EncodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("wechat: 解码 EncodingAESKey 失败: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("wechat: EncodingAESKey 解码后长度应为 32，实际为 %d", len(keyBytes))
	}

	return &App{
		config: cfg,
		aesKey: keyBytes,
	}, nil
}

// VerifyURL 处理企微回调 URL 验证（GET 请求）
// 企微会发送 GET 请求携带 msg_signature, timestamp, nonce, echostr
// 验证签名后解密 echostr 返回给企微
func (a *App) VerifyURL(msgSignature, timestamp, nonce, echostr string) (string, error) {
	// 验证消息签名
	if err := a.verifySignature(msgSignature, timestamp, nonce, echostr); err != nil {
		return "", err
	}

	// 解密 echostr
	plaintext, _, err := a.decrypt(echostr)
	if err != nil {
		return "", fmt.Errorf("wechat: 解密 echostr 失败: %w", err)
	}

	return string(plaintext), nil
}

// encryptedXMLBody 企微加密消息的 XML 结构
type encryptedXMLBody struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	Encrypt    string   `xml:"Encrypt"`
}

// DecryptMessage 解密企微推送的加密 XML 消息体
// 输入：加密的 XML 消息体（字节切片）
// 输出：解密后的明文 XML 字节切片
func (a *App) DecryptMessage(msgSignature, timestamp, nonce string, encryptedBody []byte) ([]byte, error) {
	var body encryptedXMLBody
	if err := xml.Unmarshal(encryptedBody, &body); err != nil {
		return nil, fmt.Errorf("wechat: 解析加密 XML 失败: %w", err)
	}

	// 验证消息签名
	if err := a.verifySignature(msgSignature, timestamp, nonce, body.Encrypt); err != nil {
		return nil, err
	}

	// 解密消息体
	plaintext, _, err := a.decrypt(body.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("wechat: 解密消息体失败: %w", err)
	}

	return plaintext, nil
}

// replyEncryptedXML 加密回复消息的 XML 结构
type replyEncryptedXML struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      string   `xml:"Encrypt"`
	MsgSignature string   `xml:"MsgSignature"`
	TimeStamp    string   `xml:"TimeStamp"`
	Nonce        string   `xml:"Nonce"`
}

// EncryptReply 加密回复消息，生成符合企微格式的 XML 字符串
func (a *App) EncryptReply(replyMsg, timestamp, nonce string) (string, error) {
	encrypted, err := a.encrypt([]byte(replyMsg))
	if err != nil {
		return "", fmt.Errorf("wechat: 加密回复消息失败: %w", err)
	}

	// 计算签名
	sig := a.computeSignature(timestamp, nonce, encrypted)

	reply := replyEncryptedXML{
		Encrypt:      encrypted,
		MsgSignature: sig,
		TimeStamp:    timestamp,
		Nonce:        nonce,
	}

	out, err := xml.Marshal(reply)
	if err != nil {
		return "", fmt.Errorf("wechat: 序列化回复 XML 失败: %w", err)
	}

	return string(out), nil
}

// verifySignature 校验企微消息签名
// 签名算法：SHA1(sort(token, timestamp, nonce, encryptMsg))
func (a *App) verifySignature(msgSignature, timestamp, nonce, encryptMsg string) error {
	expected := a.computeSignature(timestamp, nonce, encryptMsg)
	if expected != msgSignature {
		return fmt.Errorf("wechat: 消息签名验证失败")
	}
	return nil
}

// computeSignature 计算消息签名
func (a *App) computeSignature(timestamp, nonce, encryptMsg string) string {
	strs := []string{a.config.Token, timestamp, nonce, encryptMsg}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// decrypt 解密企微 AES-256-CBC 加密消息
// 消息格式：16字节随机串 + 4字节消息长度(网络序) + 明文消息 + corp_id
func (a *App) decrypt(encryptedMsg string) (plaintext []byte, corpID string, err error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return nil, "", fmt.Errorf("base64 解码失败: %w", err)
	}

	block, err := aes.NewCipher(a.aesKey)
	if err != nil {
		return nil, "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, "", fmt.Errorf("密文长度不足")
	}

	// IV 使用 AES key 的前 16 字节（企微规范）
	iv := a.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除 PKCS#7 padding
	ciphertext, err = pkcs7Unpad(ciphertext)
	if err != nil {
		return nil, "", fmt.Errorf("去除 padding 失败: %w", err)
	}

	// 解析消息结构：16字节随机串 + 4字节长度 + 明文 + corp_id
	if len(ciphertext) < 20 {
		return nil, "", fmt.Errorf("解密后数据长度不足")
	}

	// 跳过前 16 字节随机串，读取 4 字节消息长度
	msgLen := binary.BigEndian.Uint32(ciphertext[16:20])
	if int(msgLen)+20 > len(ciphertext) {
		return nil, "", fmt.Errorf("消息长度字段非法")
	}

	plaintext = ciphertext[20 : 20+msgLen]
	corpID = string(ciphertext[20+msgLen:])

	return plaintext, corpID, nil
}

// encrypt 使用 AES-256-CBC 加密消息
// 消息格式：16字节随机串 + 4字节消息长度(网络序) + 明文消息 + corp_id
func (a *App) encrypt(msg []byte) (string, error) {
	// 生成 16 字节随机串
	randomBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return "", fmt.Errorf("生成随机串失败: %w", err)
	}

	// 构造明文：随机串 + 4字节消息长度 + 消息 + corp_id
	corpIDBytes := []byte(a.config.CorpID)
	msgLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBytes, uint32(len(msg)))

	plaintext := make([]byte, 0, 16+4+len(msg)+len(corpIDBytes))
	plaintext = append(plaintext, randomBytes...)
	plaintext = append(plaintext, msgLenBytes...)
	plaintext = append(plaintext, msg...)
	plaintext = append(plaintext, corpIDBytes...)

	// PKCS#7 填充
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)

	block, err := aes.NewCipher(a.aesKey)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	iv := a.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// pkcs7Pad 对数据进行 PKCS#7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padBytes := make([]byte, padding)
	for i := range padBytes {
		padBytes[i] = byte(padding)
	}
	return append(data, padBytes...)
}

// pkcs7Unpad 去除 PKCS#7 填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据为空")
	}
	padding := int(data[len(data)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("padding 值非法: %d", padding)
	}
	return data[:len(data)-padding], nil
}
