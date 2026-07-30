// file: pkg/providers/registry.go
// version: 1.1.0
// guid: 85c2d092-08c4-4009-b9ec-502f12ae3794
// last-edited: 2026-07-30

package providers

import (
	"fmt"
	"sort"
	"sync"

	"github.com/jdfalk/subtitle-manager/pkg/providers/addic7ed"
	"github.com/jdfalk/subtitle-manager/pkg/providers/animekalesi"
	"github.com/jdfalk/subtitle-manager/pkg/providers/animetosho"
	"github.com/jdfalk/subtitle-manager/pkg/providers/assrt"
	"github.com/jdfalk/subtitle-manager/pkg/providers/avistaz"
	"github.com/jdfalk/subtitle-manager/pkg/providers/betaseries"
	"github.com/jdfalk/subtitle-manager/pkg/providers/bsplayer"
	"github.com/jdfalk/subtitle-manager/pkg/providers/embedded"
	"github.com/jdfalk/subtitle-manager/pkg/providers/generic"
	"github.com/jdfalk/subtitle-manager/pkg/providers/gestdown"
	"github.com/jdfalk/subtitle-manager/pkg/providers/greeksubs"
	"github.com/jdfalk/subtitle-manager/pkg/providers/greeksubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/hdbits"
	"github.com/jdfalk/subtitle-manager/pkg/providers/hosszupuska"
	"github.com/jdfalk/subtitle-manager/pkg/providers/karagarga"
	"github.com/jdfalk/subtitle-manager/pkg/providers/ktuvit"
	"github.com/jdfalk/subtitle-manager/pkg/providers/legendasdivx"
	"github.com/jdfalk/subtitle-manager/pkg/providers/legendasnet"
	"github.com/jdfalk/subtitle-manager/pkg/providers/napiprojekt"
	"github.com/jdfalk/subtitle-manager/pkg/providers/napisy24"
	"github.com/jdfalk/subtitle-manager/pkg/providers/nekur"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitlescom"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitlesvip"
	"github.com/jdfalk/subtitle-manager/pkg/providers/podnapisi"
	"github.com/jdfalk/subtitle-manager/pkg/providers/regielive"
	"github.com/jdfalk/subtitle-manager/pkg/providers/soustitres"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subdivx"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subf2m"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subs4free"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subs4series"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subscene"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subscenter"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subssabbz"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subsunacs"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subsynchro"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subtitrarinoi"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subtitriidlv"
	"github.com/jdfalk/subtitle-manager/pkg/providers/subtitulamos"
	"github.com/jdfalk/subtitle-manager/pkg/providers/supersubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/titlovi"
	"github.com/jdfalk/subtitle-manager/pkg/providers/titrariro"
	"github.com/jdfalk/subtitle-manager/pkg/providers/titulky"
	"github.com/jdfalk/subtitle-manager/pkg/providers/turkcealtyazi"
	"github.com/jdfalk/subtitle-manager/pkg/providers/tusubtitulo"
	"github.com/jdfalk/subtitle-manager/pkg/providers/tvsubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/whisper"
	"github.com/jdfalk/subtitle-manager/pkg/providers/wizdom"
	"github.com/jdfalk/subtitle-manager/pkg/providers/xsubs"
	"github.com/jdfalk/subtitle-manager/pkg/providers/yavka"
	"github.com/jdfalk/subtitle-manager/pkg/providers/yifysubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/zimuku"
)

var factories = map[string]func() Provider{
	"addic7ed":         func() Provider { return addic7ed.New() },
	"animekalesi":      func() Provider { return animekalesi.New() },
	"animetosho":       func() Provider { return animetosho.New() },
	"assrt":            func() Provider { return assrt.New() },
	"avistaz":          func() Provider { return avistaz.New() },
	"betaseries":       func() Provider { return betaseries.New() },
	"bsplayer":         func() Provider { return bsplayer.New() },
	"embedded":         func() Provider { return embedded.New() },
	"gestdown":         func() Provider { return gestdown.New() },
	"generic":          func() Provider { return generic.New() },
	"greeksubs":        func() Provider { return greeksubs.New() },
	"greeksubtitles":   func() Provider { return greeksubtitles.New() },
	"hdbits":           func() Provider { return hdbits.New() },
	"hosszupuska":      func() Provider { return hosszupuska.New() },
	"karagarga":        func() Provider { return karagarga.New() },
	"ktuvit":           func() Provider { return ktuvit.New() },
	"legendasdivx":     func() Provider { return legendasdivx.New() },
	"legendasnet":      func() Provider { return legendasnet.New() },
	"napiprojekt":      func() Provider { return napiprojekt.New() },
	"napisy24":         func() Provider { return napisy24.New() },
	"nekur":            func() Provider { return nekur.New() },
	"opensubtitlescom": func() Provider { return opensubtitlescom.New() },
	"opensubtitlesvip": func() Provider { return opensubtitlesvip.New() },
	"podnapisi":        func() Provider { return podnapisi.New() },
	"regielive":        func() Provider { return regielive.New() },
	"soustitres":       func() Provider { return soustitres.New() },
	"subdivx":          func() Provider { return subdivx.New() },
	"subf2m":           func() Provider { return subf2m.New() },
	"subscene":         func() Provider { return subscene.New() },
	"subs4free":        func() Provider { return subs4free.New() },
	"subs4series":      func() Provider { return subs4series.New() },
	"subscenter":       func() Provider { return subscenter.New() },
	"subssabbz":        func() Provider { return subssabbz.New() },
	"subsunacs":        func() Provider { return subsunacs.New() },
	"subsynchro":       func() Provider { return subsynchro.New() },
	"subtitrarinoi":    func() Provider { return subtitrarinoi.New() },
	"subtitriidlv":     func() Provider { return subtitriidlv.New() },
	"subtitulamos":     func() Provider { return subtitulamos.New() },
	"supersubtitles":   func() Provider { return supersubtitles.New() },
	"titlovi":          func() Provider { return titlovi.New() },
	"titrariro":        func() Provider { return titrariro.New() },
	"titulky":          func() Provider { return titulky.New() },
	"turkcealtyazi":    func() Provider { return turkcealtyazi.New() },
	"tusubtitulo":      func() Provider { return tusubtitulo.New() },
	"tvsubtitles":      func() Provider { return tvsubtitles.New() },
	"whisper":          func() Provider { return whisper.New() },
	"wizdom":           func() Provider { return wizdom.New() },
	"xsubs":            func() Provider { return xsubs.New() },
	"yavka":            func() Provider { return yavka.New() },
	"yifysubtitles":    func() Provider { return yifysubtitles.New() },
	"zimuku":           func() Provider { return zimuku.New() },
}

// factoriesMu guards factories.
//
// The map was unguarded while providers were only ever consulted one at a
// time. FetchFromAll now queries several concurrently, so Get is called from
// multiple goroutines and the map needs a lock — concurrent reads alone are
// safe, but RegisterFactory writes it, and Go's race detector (correctly)
// flags the combination.
var factoriesMu sync.RWMutex

// RegisterFactory adds a provider factory to the registry. Primarily used in tests
// to register mock providers.
func RegisterFactory(name string, f func() Provider) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}

// Get returns a provider by name.
func Get(name, _ string) (Provider, error) {
	if name == "opensubtitles" {
		return opensubtitles.New(""), nil
	}
	factoriesMu.RLock()
	f, ok := factories[name]
	factoriesMu.RUnlock()
	if ok {
		return f(), nil
	}
	return nil, fmt.Errorf("unknown provider %s", name)
}

// All returns the list of known provider names in alphabetical order.
func All() []string {
	factoriesMu.RLock()
	names := make([]string, 0, len(factories)+1)
	names = append(names, "opensubtitles")
	for n := range factories {
		names = append(names, n)
	}
	factoriesMu.RUnlock()
	sort.Strings(names)
	return names
}
