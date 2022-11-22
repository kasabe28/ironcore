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

package volume

import (
	"context"
	"fmt"
	"os"

	ori "github.com/onmetal/onmetal-api/ori/apis/volume/v1alpha1"
	clicommon "github.com/onmetal/onmetal-api/orictl/cli/common"
	"github.com/onmetal/onmetal-api/orictl/cmd/orictl-volume/orictlvolume/common"
	"github.com/onmetal/onmetal-api/orictl/decoder"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	Filename string
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Filename, "filename", "f", o.Filename, "Path to a file to read.")
}

func Command(streams clicommon.Streams, clientFactory common.ClientFactory) *cobra.Command {
	var opts Options

	cmd := &cobra.Command{
		Use:     "volume",
		Aliases: common.VolumeAliases,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := ctrl.LoggerFrom(ctx)

			client, cleanup, err := clientFactory.New()
			if err != nil {
				return err
			}
			defer func() {
				if err := cleanup(); err != nil {
					log.Error(err, "Error cleaning up")
				}
			}()

			return Run(ctx, streams, client, opts)
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func Run(ctx context.Context, streams clicommon.Streams, client ori.VolumeRuntimeClient, opts Options) error {
	data, err := clicommon.ReadFileOrReader(opts.Filename, os.Stdin)
	if err != nil {
		return err
	}

	volumeConfig := &ori.VolumeConfig{}
	if err := decoder.Decode(data, volumeConfig); err != nil {
		return err
	}

	res, err := client.CreateVolume(ctx, &ori.CreateVolumeRequest{Config: volumeConfig})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(streams.Out, res.Volume.Id)
	return nil
}
