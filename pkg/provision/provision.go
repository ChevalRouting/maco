package provision

type File struct {
	Name    string
	Content string
}

type Params struct {
	Hostname      string
	MAC           string
	Addresses     []string
	Gateway       string
	Nameservers   []string
	NetworkConfig string
}

type Delivery struct {
	SeedPath     string
	IgnitionPath string
}

type Provisioner interface {
	Name() string
	Render(p Params) ([]File, error)
	Deliver(dir string, files []File) (Delivery, error)
}

var registry = map[string]Provisioner{}

func Register(p Provisioner) {
	registry[p.Name()] = p
}

func Get(name string) Provisioner {
	return registry[name]
}
