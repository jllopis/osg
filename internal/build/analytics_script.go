package build

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateAnalyticsScript writes a lightweight analytics tracking script
// to publicDir/js/analytics.js if analytics is enabled in the config.
func GenerateAnalyticsScript(publicDir, apiURL string) error {
	dir := filepath.Join(publicDir, "js")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir js: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "analytics.js"), []byte(analyticsJS(apiURL)), 0o644)
}

func analyticsJS(apiURL string) string {
	return fmt.Sprintf(`(function(){
  "use strict";
  if(navigator.doNotTrack==="1"||window.doNotTrack==="1")return;
  var api=%q;
  var ref=document.referrer||"";
  var path=location.pathname;
  try{
    var body=JSON.stringify({path:path,referrer:ref});
    if(navigator.sendBeacon){
      navigator.sendBeacon(api+"/api/v1/analytics/hit",new Blob([body],{type:"application/json"}));
    }else{
      var x=new XMLHttpRequest();
      x.open("POST",api+"/api/v1/analytics/hit",true);
      x.setRequestHeader("Content-Type","application/json");
      x.send(body);
    }
  }catch(e){}
})();
`, apiURL)
}
