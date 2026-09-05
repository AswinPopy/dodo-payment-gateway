package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	HeaderID        = "Webhook-Id"
	HeaderTimestamp = "Webhook-Timestamp"
	HeaderSignature = "Webhook-Signature"

	// ReplayWindow rejects signed payloads whose timestamp
	// is older or newer than this skew.
	ReplayWindow = 5 * time.Minute

	DefaultPSPSecret = "whsec_mock_psp_dev"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrReplay           = errors.New("webhook timestamp outside replay window")
)

func NewSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

// Sign produces HMAC-SHA256(secret, eventID.unixTimestamp.rawBody).
func Sign(secret string, eventID string, timestamp time.Time, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload(eventID, timestamp.Unix(), body)))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func Verify(
	secret string,
	eventID string,
	timestampHeader string,
	signature string,
	body []byte,
	now time.Time,
) error {
	unixTs, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}

	skew := now.Sub(time.Unix(unixTs, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > ReplayWindow {
		return ErrReplay
	}

	expected := Sign(secret, eventID, time.Unix(unixTs, 0), body)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}

	return nil
}

func signedPayload(eventID string, unixTs int64, body []byte) string {
	return fmt.Sprintf("%s.%d.%s", eventID, unixTs, body)
}
