// Copyright 2010 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package oauth

import (
	"testing"
	"bytes"
	"github.com/garyburd/twister/web"
)

var signatureTests = []struct {
	method            string
	url               string
	param             web.ParamMap
	base              string
	clientCredentials Credentials
	credentials       Credentials
	sig               string
}{
	{
		"GeT",
		"hTtp://pHotos.example.net/photos",
		web.NewParamMap(
			"oauth_consumer_key", "dpf43f3p2l4k3l03",
			"oauth_token", "nnch734d00sl2jdk",
			"oauth_nonce", "kllo9940pd9333jh",
			"oauth_timestamp", "1191242096",
			"oauth_signature_method", "HMAC-SHA1",
			"oauth_version", "1.0",
			"size", "original",
			"file", "vacation.jpg"),
		"GET&http%3A%2F%2Fphotos.example.net%2Fphotos&file%3Dvacation.jpg%26oauth_consumer_key%3Ddpf43f3p2l4k3l03%26oauth_nonce%3Dkllo9940pd9333jh%26oauth_signature_method%3DHMAC-SHA1%26oauth_timestamp%3D1191242096%26oauth_token%3Dnnch734d00sl2jdk%26oauth_version%3D1.0%26size%3Doriginal",
		Credentials{"dpf43f3p2l4k3l03", "kd94hf93k423kf44"},
		Credentials{"kd94hf93k423kf44", "pfkkdhi9sl3r4s00"},
		"tR3+Ty81lMeYAr/Fid0kMTYa/WM="},
	{
		"GET",
		"http://PHOTOS.example.net:8001/Photos",
		web.NewParamMap(
			"oauth_consumer_key", "dpf43f3++p+#2l4k3l03",
			"oauth_token", "nnch734d(0)0sl2jdk",
			"oauth_nonce", "kllo~9940~pd9333jh",
			"oauth_timestamp", "1191242096",
			"oauth_signature_method", "HMAC-SHA1",
			"oauth_version", "1.0",
			"photo size", "300%",
			"title", "Back of $100 Dollars Bill"),
		"GET&http%3A%2F%2Fphotos.example.net%3A8001%2FPhotos&oauth_consumer_key%3Ddpf43f3%252B%252Bp%252B%25232l4k3l03%26oauth_nonce%3Dkllo~9940~pd9333jh%26oauth_signature_method%3DHMAC-SHA1%26oauth_timestamp%3D1191242096%26oauth_token%3Dnnch734d%25280%25290sl2jdk%26oauth_version%3D1.0%26photo%2520size%3D300%2525%26title%3DBack%2520of%2520%2524100%2520Dollars%2520Bill",
		Credentials{"dpf43f3++p+#2l4k3l03", "kd9@4h%%4f93k423kf44"},
		Credentials{"nnch734d(0)0sl2jdk", "pfkkd#hi9_sl-3r=4s00"},
		"tTFyqivhutHiglPvmyilZlHm5Uk="},
}

func TestSignature(t *testing.T) {
	for _, st := range signatureTests {
		var buf bytes.Buffer
		writeBaseString(&buf, st.method, st.url, st.param)
		base := buf.String()
		if base != st.base {
			t.Errorf("base string for %s %s = %q, want %q", st.method, st.url, st.base, base)
		}
		sig := signature(&st.clientCredentials, &st.credentials, st.method, st.url, st.param)
		if sig != st.sig {
			t.Errorf("signature for %s %s = %q, want %q", st.method, st.url, st.sig, sig)
		}
	}
}
