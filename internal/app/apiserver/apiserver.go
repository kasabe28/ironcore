// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/onmetal/onmetal-api/internal/admission/plugin/volumeresizepolicy"

	computev1beta1 "github.com/onmetal/onmetal-api/api/compute/v1beta1"
	corev1beta1 "github.com/onmetal/onmetal-api/api/core/v1beta1"
	ipamv1beta1 "github.com/onmetal/onmetal-api/api/ipam/v1beta1"
	networkingv1beta1 "github.com/onmetal/onmetal-api/api/networking/v1beta1"
	storagev1beta1 "github.com/onmetal/onmetal-api/api/storage/v1beta1"
	"github.com/onmetal/onmetal-api/client-go/informers"
	clientset "github.com/onmetal/onmetal-api/client-go/onmetalapi"
	onmetalopenapi "github.com/onmetal/onmetal-api/client-go/openapi"
	onmetalapiinitializer "github.com/onmetal/onmetal-api/internal/admission/initializer"
	"github.com/onmetal/onmetal-api/internal/admission/plugin/machinevolumedevices"
	"github.com/onmetal/onmetal-api/internal/admission/plugin/resourcequota"
	"github.com/onmetal/onmetal-api/internal/api"
	"github.com/onmetal/onmetal-api/internal/apis/compute"
	"github.com/onmetal/onmetal-api/internal/apiserver"
	"github.com/onmetal/onmetal-api/internal/machinepoollet/client"
	"github.com/onmetal/onmetal-api/internal/quota/evaluator/onmetal"
	apiequality "github.com/onmetal/onmetal-api/utils/equality"
	"github.com/onmetal/onmetal-api/utils/quota"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	netutils "k8s.io/utils/net"
)

const defaultEtcdPathPrefix = "/registry/onmetal.de"

func init() {
	utilruntime.Must(apiequality.AddFuncs(equality.Semantic))
}

func NewResourceConfig() *serverstorage.ResourceConfig {
	cfg := serverstorage.NewResourceConfig()
	cfg.EnableVersions(
		computev1beta1.SchemeGroupVersion,
		corev1beta1.SchemeGroupVersion,
		storagev1beta1.SchemeGroupVersion,
		networkingv1beta1.SchemeGroupVersion,
		ipamv1beta1.SchemeGroupVersion,
	)
	return cfg
}

type OnmetalAPIServerOptions struct {
	RecommendedOptions   *genericoptions.RecommendedOptions
	MachinePoolletConfig client.MachinePoolletClientConfig

	SharedInformerFactory informers.SharedInformerFactory
}

func (o *OnmetalAPIServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)

	// machinepoollet related flags:
	fs.StringSliceVar(&o.MachinePoolletConfig.PreferredAddressTypes, "machinepoollet-preferred-address-types", o.MachinePoolletConfig.PreferredAddressTypes,
		"List of the preferred MachinePoolAddressTypes to use for machinepoollet connections.")

	fs.DurationVar(&o.MachinePoolletConfig.HTTPTimeout, "machinepoollet-timeout", o.MachinePoolletConfig.HTTPTimeout,
		"Timeout for machinepoollet operations.")

	fs.StringVar(&o.MachinePoolletConfig.CertFile, "machinepoollet-client-certificate", o.MachinePoolletConfig.CertFile,
		"Path to a client cert file for TLS.")

	fs.StringVar(&o.MachinePoolletConfig.KeyFile, "machinepoollet-client-key", o.MachinePoolletConfig.KeyFile,
		"Path to a client key file for TLS.")

	fs.StringVar(&o.MachinePoolletConfig.CAFile, "machinepoollet-certificate-authority", o.MachinePoolletConfig.CAFile,
		"Path to a cert file for the certificate authority.")
}

func NewOnmetalAPIServerOptions() *OnmetalAPIServerOptions {
	o := &OnmetalAPIServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			api.Codecs.LegacyCodec(
				computev1beta1.SchemeGroupVersion,
				corev1beta1.SchemeGroupVersion,
				storagev1beta1.SchemeGroupVersion,
				networkingv1beta1.SchemeGroupVersion,
				ipamv1beta1.SchemeGroupVersion,
			),
		),
		MachinePoolletConfig: client.MachinePoolletClientConfig{
			Port:         12319,
			ReadOnlyPort: 12320,
			PreferredAddressTypes: []string{
				string(compute.MachinePoolHostName),

				// internal, preferring DNS if reported
				string(compute.MachinePoolInternalDNS),
				string(compute.MachinePoolInternalIP),

				// external, preferring DNS if reported
				string(compute.MachinePoolExternalDNS),
				string(compute.MachinePoolExternalIP),
			},
			HTTPTimeout: time.Duration(5) * time.Second,
		},
	}
	o.RecommendedOptions.Etcd.StorageConfig.EncodeVersioner = runtime.NewMultiGroupVersioner(
		computev1beta1.SchemeGroupVersion,
		schema.GroupKind{Group: computev1beta1.SchemeGroupVersion.Group},
		schema.GroupKind{Group: corev1beta1.SchemeGroupVersion.Group},
		schema.GroupKind{Group: storagev1beta1.SchemeGroupVersion.Group},
		schema.GroupKind{Group: networkingv1beta1.SchemeGroupVersion.Group},
		schema.GroupKind{Group: ipamv1beta1.SchemeGroupVersion.Group},
	)
	return o
}

func NewCommandStartOnmetalAPIServer(ctx context.Context, defaults *OnmetalAPIServerOptions) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch an onmetal-api API server",
		Long:  "Launch an onmetal-api API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.Run(ctx); err != nil {
				return err
			}
			return nil
		},
	}

	o.AddFlags(cmd.Flags())
	utilfeature.DefaultMutableFeatureGate.AddFlag(cmd.Flags())

	return cmd
}

func (o *OnmetalAPIServerOptions) Validate(args []string) error {
	var errors []error
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o *OnmetalAPIServerOptions) Complete() error {
	machinevolumedevices.Register(o.RecommendedOptions.Admission.Plugins)
	resourcequota.Register(o.RecommendedOptions.Admission.Plugins)
	volumeresizepolicy.Register(o.RecommendedOptions.Admission.Plugins)

	o.RecommendedOptions.Admission.RecommendedPluginOrder = append(
		o.RecommendedOptions.Admission.RecommendedPluginOrder,
		machinevolumedevices.PluginName,
		resourcequota.PluginName,
		volumeresizepolicy.PluginName,
	)

	return nil
}

func (o *OnmetalAPIServerOptions) Config() (*apiserver.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %w", err)
	}

	o.RecommendedOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)

	o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
		onmetalClient, err := clientset.NewForConfig(c.LoopbackClientConfig)
		if err != nil {
			return nil, err
		}

		informerFactory := informers.NewSharedInformerFactory(onmetalClient, c.LoopbackClientConfig.Timeout)
		o.SharedInformerFactory = informerFactory

		quotaRegistry := quota.NewRegistry(api.Scheme)
		if err := quota.AddAllToRegistry(quotaRegistry, onmetal.NewEvaluatorsForAdmission(onmetalClient, informerFactory)); err != nil {
			return nil, fmt.Errorf("error initializing quota registry: %w", err)
		}

		genericInitializer := onmetalapiinitializer.New(onmetalClient, informerFactory, quotaRegistry)

		return []admission.PluginInitializer{
			genericInitializer,
		}, nil
	}

	serverConfig := genericapiserver.NewRecommendedConfig(api.Codecs)

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(onmetalopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(api.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "onmetal-api"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	if utilfeature.DefaultFeatureGate.Enabled(features.OpenAPIV3) {
		serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(onmetalopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(api.Scheme))
		serverConfig.OpenAPIV3Config.Info.Title = "onmetal-api"
		serverConfig.OpenAPIV3Config.Info.Version = "0.1"
	}

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	apiResourceConfig := NewResourceConfig()

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			APIResourceConfigSource: apiResourceConfig,
			MachinePoolletConfig:    o.MachinePoolletConfig,
		},
	}

	if config.GenericConfig.EgressSelector != nil {
		// Use the config.GenericConfig.EgressSelector lookup to find the dialer to connect to the machinepoollet
		config.ExtraConfig.MachinePoolletConfig.Lookup = config.GenericConfig.EgressSelector.Lookup
	}

	return config, nil
}

func (o *OnmetalAPIServerOptions) Run(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHookOrDie("start-onmetal-api-server-informers", func(context genericapiserver.PostStartHookContext) error {
		config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
		o.SharedInformerFactory.Start(context.StopCh)
		return nil
	})

	return server.GenericAPIServer.PrepareRun().Run(ctx.Done())
}
