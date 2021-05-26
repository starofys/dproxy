package dproxy

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/fanyang01/radix"
	"gopkg.in/fsnotify.v1"
)

// Memory representation for proxy.json
//type KeyValue struct {
//	Name string `json:"name"`
//	Tunnels map[string]string `json:"tunnels"`
//}

type ProxyPass struct {
	TargetURL    *url.URL
	Target       string `json:"target" yaml:"target"`
	ChangeOrigin bool   `json:"changeOrigin" yaml:"changeOrigin"`
}

type TunnelItem struct {
	Name   string `json:"name" yaml:"name"`
	Target string `json:"target" yaml:"target"`
	Local  string `json:"local" yaml:"local"`
}

type ConfigFile struct {
	// private file file
	PrivateKey string `json:"id_rsa" yaml:"id_rsa"`
	// local addr to listen and serve, default is 127.0.0.1:1315
	// LocalSmartServer string `json:"local_smart"`
	// local addr to listen and serve, default is 127.0.0.1:1316
	LocalNormalServer string `json:"local_normal" yaml:"local_normal"`
	LocalSocketServer string `json:"local_socket" yaml:"local_socket"`
	// remote addr to connect, e.g. ssh://user@linode.my:22
	RemoteServer string `json:"remote" yaml:"remote"`
	// direct to proxy dial timeout
	ShouldProxyTimeoutMS int `json:"timeout_ms" yaml:"timeout_ms"`
	// blocked host list
	BlockedList []string `json:"proxy" yaml:"proxy1"`

	ForwardMap map[string]string `json:"forward" yaml:"forward"`

	TunnelMap []TunnelItem `json:"tunnel" yaml:"tunnel"`

	ProxyPassMap map[string]map[string]ProxyPass `json:"proxy_pass" yaml:"proxy_pass"`
}

// Load file from path
func NewConfigFile(path string) (self *ConfigFile, err error) {
	self = &ConfigFile{}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(buf, self)
	if err != nil {
		return
	}
	self.PrivateKey = os.ExpandEnv(self.PrivateKey)
	sort.Strings(self.BlockedList)
	return
}

func (cfg *Config) InitIpFilter() {
	cfg.dt = radix.NewPatternTrie()
	for i, v := range cfg.File.BlockedList {
		cfg.dt.Add(v, i+1)
	}
}

// test whether host is in blocked list or not
func (self *ConfigFile) Blocked(host string) bool {
	i := sort.SearchStrings(self.BlockedList, host)
	return i < len(self.BlockedList) && self.BlockedList[i] == host
}

// Provide global config for dproxy
type Config struct {
	// file path
	Path string
	// config file content
	File *ConfigFile
	// File wather
	Watcher *fsnotify.Watcher
	// mutex for config file
	mutex  sync.RWMutex
	loaded bool
	dt     *radix.PatternTrie
}

func NewConfig(path string) (self *Config, err error) {
	// watch config file changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	self = &Config{
		Path:    os.ExpandEnv(path),
		Watcher: watcher,
	}
	err = self.Load()
	return
}

func (cfg *Config) Reload() (err error) {
	file, err := NewConfigFile(cfg.Path)
	if err != nil {
		L.Printf("Reload %s failed: %s\n", cfg.Path, err)
	} else {
		L.Printf("Reload %s\n", cfg.Path)
		cfg.mutex.Lock()
		cfg.File = file
		cfg.InitIpFilter()
		cfg.mutex.Unlock()
	}
	return
}

// reload config file
func (cfg *Config) Load() (err error) {
	if cfg.loaded {
		panic("can not be reload manually")
	}
	cfg.loaded = true

	// first time to load
	L.Printf("Loading: %s\n", cfg.Path)
	cfg.File, err = NewConfigFile(cfg.Path)
	if err != nil {
		return
	}

	cfg.InitIpFilter()

	// Watching the whole directory instead of the individual path.
	// Because many editors won't write to file directly, they copy
	// the original one and rename it.
	err = cfg.Watcher.Add(filepath.Dir(cfg.Path))
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case event := <-cfg.Watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write && event.Name == cfg.Path {
					cfg.Reload()
				}
			case err := <-cfg.Watcher.Errors:
				L.Printf("Watching failed: %s\n", err)
			}
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP)
	go func() {
		for s := range sc {
			if s == syscall.SIGHUP {
				cfg.Reload()
			}
		}
	}()

	return
}

// test whether host is in blocked list or not
func (cfg *Config) Blocked(host string) bool {
	cfg.mutex.RLock()
	blocked := cfg.File.Blocked(host)
	cfg.mutex.RUnlock()
	return blocked
}
func (cfg *Config) Director(req *http.Request) {
}
func (cfg *Config) ReverseProxy(req *http.Request) bool {
	cfg.mutex.RLock()
	defer cfg.mutex.RUnlock()
	proxyMap := cfg.File.ProxyPassMap[req.URL.Host]
	if proxyMap == nil {
		return false
	}
	for uri, proxyCfg := range proxyMap {
		index := strings.IndexAny(req.URL.Path, uri)
		if index == -1 {
			continue
		}
		index += len(uri)
		//matched, err := path.Match(uri, req.URL.Path)
		//if err != nil || !matched {
		//	continue
		//}

		if proxyCfg.TargetURL == nil {
			proxyCfg.TargetURL, _ = url.Parse(proxyCfg.Target)
		}

		if proxyCfg.TargetURL == nil {
			L.Printf("Invalid URL %s\n", proxyCfg.Target)
		} else {
			target := proxyCfg.TargetURL
			targetQuery := target.RawQuery
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host

			if strings.HasSuffix(proxyCfg.Target, "/") {
				req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path[index:])
			} else {
				req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
			}
			//if strings.HasSuffix(proxyCfg.Target,"/") {
			//	req.URL.Path = target.Path
			//} else {
			//	req.URL.Path = path.Join(target.Path,req.URL.Path)
			//}
			//if strings.HasSuffix(proxyCfg.Target,"/") {
			//	req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
			//} else {
			//	req.URL.Path = target.Path
			//	req.URL.RawPath = target.RawPath
			//}

			req.Host = target.Host
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			L.Printf("%s\n", req.URL.Path)
			return true
		}
		break
	}
	return false
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
