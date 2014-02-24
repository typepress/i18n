package i18n

import (
	"fmt"
	"mime"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/typepress/core/types"

	"github.com/codegangsta/martini"
)

var (
	mm   = map[string]map[string]Translator{}
	trwr = sync.RWMutex{}
)

const (
	NameOfCookieToUseLanguage = "lang"
)

/*
  Register an Translator for translate from source to destination language.
  source means abstract language.
  destination means http request head "Accept-Language". See also:
	http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4
	http://www.w3.org/Protocols/rfc2616/rfc2616-sec3.html#sec3.10
  e.g
	Register("en","zh",yourtrs)
	Register("en.mysql","zh-cn",yourtrs)
*/
func Register(source, destination string, trs Translator) {
	trwr.Lock()
	defer trwr.Unlock()
	m := mm[source]
	if m == nil {
		mm[source] = map[string]Translator{}
		m = mm[source]
	}
	m[destination] = trs
}

type Translator interface {
	Sprint(...interface{}) string
	Sprintf(string, ...interface{}) string
}

var typeTranslator = reflect.TypeOf(types.Translator(nil))

func Translate(source string) martini.Handler {
	return func(r *http.Request, c martini.Context) {
		var tr *trans
		rv := c.Get(typeTranslator)
		if rv.IsValid() {
			ts := rv.Interface().(types.Translator)
			tr = ts.(*trans)
		}
		if tr == nil {
			tr = &trans{source, sortAcceptLang(r)}
		} else if len(tr.acceptlang) == 0 {
			tr.acceptlang = sortAcceptLang(r)
			tr.source = source
		}
		c.Map(types.Translator(tr))
	}
}

type trans struct {
	source     string // source language
	acceptlang []string
}

func (tr *trans) Source(source string) {
	if tr != nil {
		tr.source = source
	}
}

func (tr *trans) Sprint(v ...interface{}) string {
	return tr.Sprintf("", v...)
}

func (tr *trans) Sprintf(format string, v ...interface{}) string {
	if len(mm) == 0 || tr == nil || len(tr.source) == 0 || len(tr.acceptlang) == 0 {
		return tr.sprintf(format, v...)
	}

	trwr.RLock()
	defer trwr.RUnlock()

	m := mm[tr.source]
	if m == nil {
		return tr.sprintf(format, v...)
	}

	var s string
	for _, ac := range tr.acceptlang {
		t := m[ac]
		if t != nil {
			if len(format) == 0 {
				s = t.Sprint(v...)
			} else {
				s = t.Sprintf(format, v...)
			}
			if len(s) != 0 {
				return s
			}
		}
	}

	return tr.sprintf(format, v...)
}

func (tr *trans) sprintf(format string, v ...interface{}) string {
	if len(format) == 0 {
		return fmt.Sprint(v...)
	}
	return fmt.Sprintf(format, v...)
}

func sortAcceptLang(r *http.Request) []string {
	var langs []string
	var al string
	ck, _ := r.Cookie(NameOfCookieToUseLanguage)
	if ck != nil {
		al = ck.Value
	}

	if al == "" {
		al = r.Header.Get("Accept-Language")
	}
	ss := strings.Split(al, ",")
	langs = []string{strings.ToLower(ss[0])}
	for i := 1; i < len(ss); i++ {
		mediatype, params, err := mime.ParseMediaType(ss[i])
		if err != nil {
			break
		}
		if params["q"] != "0" {
			langs = append(langs, mediatype)
		}
	}
	return langs
}
