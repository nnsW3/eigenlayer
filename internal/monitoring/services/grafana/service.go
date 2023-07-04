package grafana

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"text/template"

	datadir "github.com/NethermindEth/egn/internal/data"
	"github.com/NethermindEth/egn/internal/monitoring"
	"github.com/NethermindEth/egn/internal/monitoring/services/types"
)

//go:embed config
var config embed.FS

//go:embed dashboards
var dashboards embed.FS

// Verify that GrafanaService implements the ServiceAPI interface.
var _ monitoring.ServiceAPI = &GrafanaService{}

// GrafanaService implements the ServiceAPI interface for a Grafana service.
type GrafanaService struct {
	stack *datadir.MonitoringStack
}

// NewGrafana creates a new GrafanaService.
func NewGrafana() *GrafanaService {
	return &GrafanaService{}
}

// Init initializes the Grafana service with the given options.
func (g *GrafanaService) Init(opts types.ServiceOptions) error {
	g.stack = opts.Stack
	return nil
}

func (g *GrafanaService) AddTarget(endpoint string) error {
	return nil
}

func (g *GrafanaService) RemoveTarget(endpoint string) error {
	return nil
}

// DotEnv returns the dotenv variables and default values for the Grafana service.
func (g *GrafanaService) DotEnv() map[string]string {
	return dotEnv
}

// Setup sets up the Grafana service provisioning and configuration with the given dotenv values.
func (g *GrafanaService) Setup(options map[string]string) error {
	// Validate options
	promPort, ok := options["PROM_PORT"]
	if !ok {
		return fmt.Errorf("%w: %s missing in options", ErrInvalidOptions, "PROM_PORT")
	} else if promPort == "" {
		return fmt.Errorf("%w: %s can't be empty", ErrInvalidOptions, "PROM_PORT")
	}

	// Read config template
	rawTmp, err := config.ReadFile("config/prom.yml")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConfigNotFound, err)
	}
	// Load template
	tmp, err := template.New("prom.yml").Parse(string(rawTmp))
	if err != nil {
		return err
	}

	// Create config directory
	grafProvPath := filepath.Join("grafana", "provisioning")
	if err = g.stack.CreateDir(filepath.Join(grafProvPath, "datasources")); err != nil {
		return err
	}
	// Create config file
	configFile, err := g.stack.Create(filepath.Join(grafProvPath, "datasources", "prom.yml"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	// Execute template
	data := struct {
		PromEndpoint string
	}{
		PromEndpoint: fmt.Sprintf("http://%s:%s", monitoring.PrometheusServiceName, options["PROM_PORT"]),
	}
	err = tmp.Execute(configFile, data)
	if err != nil {
		return err
	}

	// Create provisioning dashboards folder
	if err = g.stack.CreateDir(filepath.Join(grafProvPath, "dashboards")); err != nil {
		return err
	}
	// Create dashboards provisioning file
	dashboardsFile, err := g.stack.Create(filepath.Join(grafProvPath, "dashboards", "dashboards.yml"))
	if err != nil {
		return err
	}
	defer dashboardsFile.Close()
	dbs, err := config.ReadFile("config/dashboards.yml")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConfigNotFound, err)
	}
	if _, err = dashboardsFile.Write(dbs); err != nil {
		return err
	}

	// Copy dashboards
	if err = g.copyDashboards(filepath.Join("grafana", "data")); err != nil {
		return err
	}

	return nil
}

// copyDashboards copy dashboards to $DATA_DIR/dashboards
func (g *GrafanaService) copyDashboards(dst string) (err error) {
	return fs.WalkDir(dashboards, "dashboards", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			dashboard, err := dashboards.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				cerr := dashboard.Close()
				if err == nil {
					err = cerr
				}
			}()
			data, err := io.ReadAll(dashboard)
			if err != nil {
				return err
			}
			if err = g.stack.WriteFile(filepath.Join(dst, path), data); err != nil {
				return err
			}
		} else {
			if err = g.stack.CreateDir(filepath.Join(dst, path)); err != nil {
				return err
			}
		}
		return nil
	})
}

