// TODO: LICENSE??

package wavefrontexporter

import (
	"errors"

	"github.com/wavefronthq/opencensus-exporter/wavefront"
	"github.com/wavefronthq/wavefront-sdk-go/senders"

	"github.com/spf13/viper"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterwrapper"
)

type wavefrontConfig struct {
	ProxyConfiguration       *senders.ProxyConfiguration  `mapstructure:"proxy,omitempty"`
	DirectConfiguration      *senders.DirectConfiguration `mapstructure:"direct_ingestion,omitempty"`
	wavefront.ServiceOptions `mapstructure:",squash"`

	EnableTracing bool `mapstructure:"enable_tracing,omitempty"`
	EnableMetrics bool `mapstructure:"enable_metrics,omitempty"`
}

// WavefrontTraceExportersFromViper unmarshals the viper and returns trace and metric consumers.
func WavefrontTraceExportersFromViper(v *viper.Viper) (tps []consumer.TraceConsumer, mps []consumer.MetricsConsumer, doneFns []func() error, err error) {
	var cfg struct {
		Wavefront *wavefrontConfig `mapstructure:"wavefront,omitempty"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}

	wc := cfg.Wavefront
	if wc == nil {
		return nil, nil, nil, nil
	}
	if !wc.EnableTracing && !wc.EnableMetrics {
		return nil, nil, nil, nil
	}

	var ws senders.Sender

	switch {
	case wc.ProxyConfiguration != nil:
		ws, err = senders.NewProxySender(wc.ProxyConfiguration)
	case wc.DirectConfiguration != nil:
		ws, err = senders.NewDirectSender(wc.DirectConfiguration)
	default:
		err = errors.New("At least one of 'proxy' or 'direct_ingestion' must be specified")
	}
	if err != nil {
		return nil, nil, nil, err
	}

	we, err := wavefront.NewExporter(ws, wavefront.WithServiceOptions(&wc.ServiceOptions))
	if err != nil {
		return nil, nil, nil, err
	}

	doneFns = append(doneFns, func() error {
		we.Stop()
		ws.Close()
		return nil
	})

	wew, err := exporterwrapper.NewExporterWrapper("wavefront", "ocservice.exporter.Wavefront.ConsumeTraceData", we)
	if err != nil {
		return nil, nil, nil, err
	}

	tps = append(tps, wew)

	// TODO: Metrics Exporter. NewExporterWrapper only creates Trace Exporter now.
	// mps = append(mps, wew)

	return
}
