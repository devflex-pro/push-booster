package sender

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

const (
	webPushRecordSize  = 4096
	p256PublicKeySize  = 65
	p256PrivateKeySize = 32
)

type WebPushClient struct {
	httpClient *http.Client
	subject    string
	ttl        int
}

type triggerPayload struct {
	TriggerID string `json:"trigger_id"`
}

type vapidClaims struct {
	Audience string `json:"aud"`
	Expires  int64  `json:"exp"`
	Subject  string `json:"sub"`
}

func NewWebPushClient(subject string, ttl int) *WebPushClient {
	if ttl == 0 {
		ttl = 60
	}
	return &WebPushClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		subject:    subject,
		ttl:        ttl,
	}
}

func (c *WebPushClient) SendTrigger(
	ctx context.Context,
	task inventory.DeliveryTask,
	key inventory.VAPIDKey,
	trigger subscribers.DeliveryTrigger,
) (*http.Response, error) {
	payload, err := json.Marshal(triggerPayload{TriggerID: trigger.TriggerID})
	if err != nil {
		return nil, fmt.Errorf("marshal trigger payload: %w", err)
	}
	body, serverPublicKey, err := encryptWebPushPayload(
		payload,
		task.P256DH,
		task.Auth,
	)
	if err != nil {
		return nil, err
	}
	token, err := vapidJWT(
		task.Endpoint,
		key.PrivateKey,
		c.subject,
	)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		task.Endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create web push request: %w", err)
	}
	req.Header.Set("Authorization", "vapid t="+token+", k="+key.PublicKey)
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", fmt.Sprintf("%d", c.ttl))
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	req.Header.Set("Crypto-Key", "dh="+base64.RawURLEncoding.EncodeToString(serverPublicKey))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send web push trigger: %w", err)
	}
	return resp, nil
}

func encryptWebPushPayload(
	payload []byte,
	userPublicKey string,
	authSecret string,
) ([]byte, []byte, error) {
	receiverPublicBytes, err := decodeBase64URL(userPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode user public key: %w", err)
	}
	authSecretBytes, err := decodeBase64URL(authSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("decode auth secret: %w", err)
	}
	curve := ecdh.P256()
	receiverPublicKey, err := curve.NewPublicKey(receiverPublicBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user public key: %w", err)
	}
	serverPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate web push server key: %w", err)
	}
	sharedSecret, err := serverPrivateKey.ECDH(receiverPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("derive web push shared secret: %w", err)
	}
	serverPublicKey := serverPrivateKey.PublicKey().Bytes()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, fmt.Errorf("generate web push salt: %w", err)
	}
	info := append([]byte("WebPush: info\x00"), receiverPublicBytes...)
	info = append(info, serverPublicKey...)
	ikm, err := hkdf.Key(
		sha256.New,
		sharedSecret,
		authSecretBytes,
		string(info),
		32,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("derive web push ikm: %w", err)
	}
	cek, err := hkdf.Key(
		sha256.New,
		ikm,
		salt,
		"Content-Encoding: aes128gcm\x00",
		16,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("derive web push cek: %w", err)
	}
	nonce, err := hkdf.Key(
		sha256.New,
		ikm,
		salt,
		"Content-Encoding: nonce\x00",
		12,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("derive web push nonce: %w", err)
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, fmt.Errorf("create web push cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create web push gcm: %w", err)
	}
	plaintext := append([]byte{}, payload...)
	plaintext = append(plaintext, 0x02)
	ciphertext := aead.Seal(
		nil,
		nonce,
		plaintext,
		nil,
	)
	header := make(
		[]byte,
		21,
		21+p256PublicKeySize+len(ciphertext),
	)
	copy(header[0:16], salt)
	binary.BigEndian.PutUint32(header[16:20], webPushRecordSize)
	header[20] = byte(len(serverPublicKey))
	header = append(header, serverPublicKey...)
	header = append(header, ciphertext...)
	return header, serverPublicKey, nil
}

func vapidJWT(
	endpoint string,
	privateKey string,
	subject string,
) (string, error) {
	audience, err := vapidAudience(endpoint)
	if err != nil {
		return "", err
	}
	privateKeyBytes, err := decodeBase64URL(privateKey)
	if err != nil {
		return "", fmt.Errorf("decode vapid private key: %w", err)
	}
	if len(privateKeyBytes) != p256PrivateKeySize {
		return "", fmt.Errorf("invalid vapid private key length %d", len(privateKeyBytes))
	}
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(privateKeyBytes)
	x, y := curve.ScalarBaseMult(privateKeyBytes)
	if x == nil || y == nil {
		return "", errors.New("derive vapid public key")
	}
	key := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y},
		D:         d,
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claimsPayload, err := json.Marshal(vapidClaims{
		Audience: audience,
		Expires:  time.Now().Add(12 * time.Hour).Unix(),
		Subject:  subject,
	})
	if err != nil {
		return "", fmt.Errorf("marshal vapid claims: %w", err)
	}
	claims := base64.RawURLEncoding.EncodeToString(claimsPayload)
	signingInput := header + "." + claims
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(
		rand.Reader,
		&key,
		digest[:],
	)
	if err != nil {
		return "", fmt.Errorf("sign vapid jwt: %w", err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func vapidAudience(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse web push endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("web push endpoint must include scheme and host")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func decodeBase64URL(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	decoded, paddedErr := base64.URLEncoding.DecodeString(value)
	if paddedErr != nil {
		return nil, err
	}
	return decoded, nil
}
