package main

import (
	"example/internal/logrus_hooks"
	"example/internal/zap_config"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log/slog"
	"os"
	"unicode/utf8"
)

func main() {
	fmt.Println("********logrus*******")
	logrusExample()
	fmt.Println("********ZAP*******")
	zapExample()
	fmt.Println("********slog*******")
	slogExample()
}

func logrusExample() {
	log := logrus.New()

	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)

	fieldsToRemove := []string{"password", "credit_card"}
	fieldsToHash := []string{"user_id"}

	removerHook := logrus_hooks.NewFieldRemoverHook(fieldsToRemove, []logrus.Level{logrus.InfoLevel, logrus.ErrorLevel})
	maskerHook := logrus_hooks.NewFieldHasherHook(fieldsToHash, []logrus.Level{logrus.InfoLevel, logrus.ErrorLevel})

	log.AddHook(removerHook)
	log.AddHook(maskerHook)

	log.WithFields(logrus.Fields{
		"username":    "testuser",
		"user_id":     "abc",
		"password":    "secretpassword",
		"credit_card": "1234-5678-9012-3456",
		"order_id":    "12345",
	}).Info("User placed an order")

	log.WithFields(logrus.Fields{
		"username":    "admin",
		"user_id":     "def",
		"password":    "adminpassword",
		"credit_card": "9876-5432-1098-7654",
		"error":       "Authentication failed",
	}).Error("Login attempt failed")

	log.WithFields(logrus.Fields{
		"username":    "admin",
		"user_id":     "def",
		"password":    "adminpassword",
		"credit_card": "9876-5432-1098-7654",
		"debug":       "Something for debug",
	}).Info("Some info")
}

func zapExample() {
	productionCfg := zap.NewProductionEncoderConfig()
	productionCfg.TimeKey = "timestamp"
	productionCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	jsonEncoder := zap_config.NewSensitiveFieldsEncoder(zapcore.EncoderConfig{}, []string{"password", "credit_card", "session_id"})

	core := zapcore.NewCore(
		jsonEncoder,
		os.Stdout,
		zap.InfoLevel,
	)

	samplingCore := zapcore.NewSampler(core, 100*1024, 10, 1024)
	logger := zap.New(samplingCore)

	defer logger.Sync() // Flush the buffer at the end of the program.

	// Example log entries
	logger.Info("User login attempt",
		zap.String("username", "testuser"),
		zap.String("password", "secretpassword"),
		zap.String("credit_card", "1234-5678-9012-3456"),
		zap.String("session_id", "abcdef123456"),
		zap.String("other_data", "some other value"),
	)
	logger.Error("Authentication failure",
		zap.String("username", "badactor"),
		zap.String("password", "wrongpassword"),
		zap.String("session_id", "ghijkl789012"),
	)
	logger.Info("Purchase",
		zap.String("credit_card", "1111-2222-3333-4444"),
		zap.Int("amount", 100),
	)
}

type User struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func (u *User) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("id=%s, first_name=%s", u.ID, firstLetter(u.FirstName)))
}

// firstLetter extracts first letter from the unicode string
func firstLetter(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	return string(r)
}

func slogExample() {
	handler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(handler)

	u := &User{
		ID:        "100500",
		FirstName: "Jan",
		LastName:  "Doe",
		Email:     "jan@example.com",
	}
	u2 := &User{
		ID:        "100500",
		FirstName: "😀Jan",
		LastName:  "Doe",
		Email:     "jan@example.com",
	}

	logger.Info("info", "user", u)
	logger.Info("info", "user2", u2)
	fmt.Println(string(u2.FirstName[0]))   // garbage, like "ð"
	fmt.Println(string(u2.FirstName[0:4])) // 😀
}
