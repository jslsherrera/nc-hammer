package suite

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/beevik/etree"
	yaml "gopkg.in/yaml.v2"
)

// Sshconfig defines a definition for the parameters required to connect to a NETCONF Agent via SSH
type Sshconfig struct {
	Hostname        string
	Port            int
	Username        string
	Password        string
	Reuseconnection bool
}

// Filter defines the parameters required to generate a subtree or xpath filter within a NETCONF Request
type Filter struct {
	Type   string
	Ns     *string `json:",omitempty" yaml:",omitempty"`
	Select string
}

// Netconf struct contains information required to construct a valid NETCONF Operation.
// Addresses are used to indicate optional content
type Netconf struct {
	Hostname  string
	Operation string
	Source    *string `json:",omitempty" yaml:",omitempty"`
	Target    *string `json:",omitempty" yaml:",omitempty"`
	Filter    *Filter `json:",omitempty" yaml:",omitempty"`
	Config    *string `json:",omitempty" yaml:",omitempty"` // used in the edit-config, starts with the top element
}

// Sleep is an action instructing the client to sleep for the period defined in duration
type Sleep struct {
	Duration int // seconds
}

// Action is a wrapper for the different actions types (netconf, sleep)
type Action struct {
	Netconf *Netconf `json:",omitempty" yaml:",omitempty"`
	Sleep   *Sleep   `json:",omitempty" yaml:",omitempty"`
}

// Block describes a list of actions and how these should treated; as an init block, sequentially or concurrently
type Block struct {
	Type    string
	Actions []Action
}

// Configs rebinds the slice of Sshconfig so that methods can be constructed against it
type Configs []Sshconfig

// IsReuseConnection iterates through the Config slice and matches on host returning whether the connection should be reused or not
func (c Configs) IsReuseConnection(hostname string) bool {
	for _, config := range c {
		if config.Hostname == hostname {
			return config.Reuseconnection
		}
	}
	return false
}

// TestSuite is the top level struct for the yaml document definition
type TestSuite struct {
	File       string `json:"-" yaml:"-"`
	Iterations int
	Clients    int
	Rampup     int
	Configs    Configs
	Blocks     []Block
}

// NewTestSuite returns an TestSuite initialized from a yaml file
func NewTestSuite(file string) (*TestSuite, error) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var ts TestSuite
	err = yaml.Unmarshal(yamlFile, &ts)
	if err == nil {
		err = validateTestSuite(&ts)
	}
	ts.File = file
	return &ts, err
}

// ToXMLString generates a XML representation of the information provided in the Netconf section of the TestSuite
func (n *Netconf) ToXMLString() (string, error) {
	doc := etree.NewDocument()
	operation := doc.CreateElement(n.Operation)
	switch n.Operation {
	case "get-config":
		source := operation.CreateElement("source")
		if n.Source != nil {
			source.CreateElement(*n.Source)
		} else {
			source.CreateElement("running")
		}
		addFilterIfPresent(n, operation)
	case "get":
		addFilterIfPresent(n, operation)
	case "edit-config":
		source := operation.CreateElement("target")
		if n.Target != nil {
			source.CreateElement(*n.Target)
		} else {
			source.CreateElement("running")
		}
		config := operation.CreateElement("config")
		if n.Config != nil {
			inner := etree.NewDocument()
			err := inner.ReadFromString(*n.Config)
			if err != nil {
				log.Println("Config data is not valid xml")
			}
			config.AddChild(inner.Root().Copy())
		}
	default:
		return "", errors.New(n.Operation + " is not a supported operation")

	}
	return doc.WriteToString()
}

func addFilterIfPresent(n *Netconf, operation *etree.Element) {
	if n.Filter != nil {
		filter := operation.CreateElement("filter")
		filter.CreateAttr("type", n.Filter.Type)
		top := filter.CreateElement("top")
		if n.Filter.Ns != nil {
			top.CreateAttr("xmlns", *n.Filter.Ns)
		}
		//  https://github.com/beevik/etree/issues/49
		inner := etree.NewDocument()
		err := inner.ReadFromString(n.Filter.Select)
		if err != nil {
			log.Println("Filter Select is not valid xml")
		}
		top.AddChild(inner.Root().Copy())
	}
}

// GetConfig returns the connection information for a specific host
func (ts *TestSuite) GetConfig(hostname string) *Sshconfig {
	for _, config := range ts.Configs {
		if config.Hostname == hostname {
			return &config
		}
	}
	return nil
}

// GetInitBlock returns an init block if defined in the TestSuite
func (ts *TestSuite) GetInitBlock() *Block {
	for _, block := range ts.Blocks {
		if block.Type == "init" {
			return &block
		}
	}
	return nil
}

func validateTestSuite(ts *TestSuite) error {
	if len(ts.Configs) == 0 {
		return errors.New("Testsuite should contain at least one SSH Config section")
	}
	for _, sshconfig := range ts.Configs {
		if sshconfig.Hostname == "" {
			return errors.New("ssh config: hostname cannot be empty")
		}
		if sshconfig.Username == "" {
			return errors.New("ssh config: username cannot be empty")
		}
		if sshconfig.Password == "" {
			return errors.New("ssh config: password cannot be empty")
		}
	}
	for _, block := range ts.Blocks {
		for _, action := range block.Actions {
			if action.Netconf != nil {
				if action.Netconf.Operation == "" {
					return errors.New("netconf: operation cannot be empty")
				}
			}
		}
	}
	return nil
}
