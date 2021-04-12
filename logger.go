package vkbot

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// Logger default zap logger
var Logger *zap.Logger

func init() {
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	Logger = zap.New(core)
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func logInternalErrorOr(msg string, err error) {
	switch err.(type) {
	case *internalError: {
		err := err.(*internalError)
		Logger.Error(msg,
			zap.Error(err),
			zap.String("stack", err.StackTrace),
			zap.Reflect("misc", err.Misc))
		return
	}
	}
	Logger.Error(msg, zap.Error(err))
}
