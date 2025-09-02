package i18n

import (
	"blueprint/config"
	"github.com/kataras/i18n"
 
)

type Lang struct {
	I18n *i18n.I18n
}

// Wrap the lang translate package
func New(cfg *config.Config, languages ... string) (*Lang, error) {		
	// in production this should be diffrent 	
	glob :=i18n.Glob(cfg.Setting.LocalPath)
	new,_ := i18n.New(glob)

	lang := &Lang{
		I18n: new,
	}
	
	return lang, nil
	//return i18n.New(glob,languages...)
}

func (t *Lang) Tr(lang string,format string) string {
	return i18n.Tr(lang,format)
}







