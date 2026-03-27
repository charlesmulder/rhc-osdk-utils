package logging

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	zzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	pgm "github.com/redhatinsights/platform-go-middlewares/v2/logging/cloudwatch"
)

// buildCore creates the zapcore.Core with console and optional CloudWatch outputs
func buildCore(disableCloudwatch bool, levelEnabler zapcore.LevelEnabler) (zapcore.Core, error) {
	consoleOutput := zapcore.Lock(os.Stdout)
	consoleEncoder := zapcore.NewJSONEncoder(zzap.NewProductionEncoderConfig())

	var core zapcore.Core
	var err error

	key := os.Getenv("AWS_CW_KEY")
	secret := os.Getenv("AWS_CW_SECRET")
	group := os.Getenv("AWS_CW_LOG_GROUP")
	stream, err := os.Hostname()
	if err != nil {
		stream = "undefined"
	}
	region := os.Getenv("AWS_CW_REGION")

	endpoint := os.Getenv("AWS_CW_ENDPOINT")

	if !disableCloudwatch && key != "" {
		cred := credentials.NewStaticCredentials(key, secret, "")
		cfg := aws.NewConfig().WithRegion(region).WithCredentials(cred)
		if endpoint != "" {
			cfg = cfg.WithEndpoint(endpoint)
		}
		bw, err := pgm.NewBatchWriterWithDuration(group, stream, cfg, time.Second*5)

		if err != nil {
			return nil, err
		}

		// NewZapWriteSyncer wraps *BatchWriter to satisfy zapcore.WriteSyncer,
		// delegating Sync() to Flush() so that logger.Sync() at shutdown and on
		// Fatal/Panic log events drains the batch buffer to CloudWatch.
		cwLogger := pgm.NewZapWriteSyncer(bw)
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleOutput, levelEnabler),
			zapcore.NewCore(consoleEncoder, cwLogger, levelEnabler),
		)
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleOutput, levelEnabler),
		)
	}

	return core, err
}

// SetupLogging sets up a logger that accepts all log levels
func SetupLogging(disableCloudwatch bool) (*zzap.Logger, error) {
	fn := zzap.LevelEnablerFunc(func(_ zapcore.Level) bool {
		return true
	})

	core, err := buildCore(disableCloudwatch, fn)
	if err != nil {
		return nil, err
	}

	logger := zzap.New(core)

	return logger, err
}

// SetupLoggingWithLevel sets up a logger that filters logs below the specified level
func SetupLoggingWithLevel(disableCloudwatch bool, level int8) (*zzap.Logger, error) {
	fn := zzap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.Level(level)
	})

	core, err := buildCore(disableCloudwatch, fn)
	if err != nil {
		return nil, err
	}

	logger := zzap.New(core)

	return logger, err
}
