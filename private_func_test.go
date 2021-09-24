package middleware

import (
	"testing"
)

type InputForParseEndpoint struct {
	name      string
	input     string
	canonical string
	folders   []folder
}

func TestParseEndpoint(t *testing.T) {
	inputs := []InputForParseEndpoint{
		{
			name:      "homepage",
			input:     "/",
			canonical: "/",
			folders:   []folder{"", ""},
		},
		{
			name:      "homepage_with_extra_slashes",
			input:     "////",
			canonical: "/",
			folders:   []folder{"", ""},
		},
		{
			name:      "simple_usr",
			input:     "/usr",
			canonical: "/usr",
			folders:   []folder{"", "usr"},
		},
		{
			name:      "simple_usr_local",
			input:     "/usr/local",
			canonical: "/usr/local",
			folders:   []folder{"", "usr", "local"},
		},
		{
			name:      "folders_with_trailing_slash",
			input:     "/hello/world////",
			canonical: "/hello/world",
			folders:   []folder{"", "hello", "world"},
		},
		{
			name:      "simple_usr_local_etc",
			input:     "/usr/local/etc",
			canonical: "/usr/local/etc",
			folders:   []folder{"", "usr", "local", "etc"},
		},
		{
			name:      "simple_usr_local_etc_openssl",
			input:     "/usr/local/etc/openssl",
			canonical: "/usr/local/etc/openssl",
			folders:   []folder{"", "usr", "local", "etc", "openssl"},
		},
		{
			name:      "simple_usr_local_etc_openssl_cert.pem",
			input:     "/usr/local/etc/openssl/cert.pem",
			canonical: "/usr/local/etc/openssl/cert.pem",
			folders:   []folder{"", "usr", "local", "etc", "openssl", "cert.pem"},
		},
		{
			name:      "folders_with_extra_slashes",
			input:     "////usr////local////etc////openssl////cert.pem",
			canonical: "/usr/local/etc/openssl/cert.pem",
			folders:   []folder{"", "usr", "local", "etc", "openssl", "cert.pem"},
		},
		{
			name:      "dynamic_parameters",
			input:     "/usr/local/:group/:package/cert.pem",
			canonical: "/usr/local/:group/:package/cert.pem",
			folders:   []folder{"", "usr", "local", ":group", ":package", "cert.pem"},
		},
		{
			name:      "dynamic_parameters_and_asterisk",
			input:     "/usr/local/:group/:package/*",
			canonical: "/usr/local/:group/:package/*",
			folders:   []folder{"", "usr", "local", ":group", ":package", "*"},
		},
		{
			name:      "asterisk_with_extra_folders",
			input:     "/usr/local/:group/:package/*/hello/world",
			canonical: "/usr/local/:group/:package/*",
			folders:   []folder{"", "usr", "local", ":group", ":package", "*"},
		},
		{
			name:      "2_folders_and_an_asterisk",
			input:     "/hello/world/*/how/are/you",
			canonical: "/hello/world/*",
			folders:   []folder{"", "hello", "world", "*"},
		},
	}

	for _, x := range inputs {
		t.Run(x.name, func(t *testing.T) {
			end := parseEndpoint(x.input, nil)

			if end.String() != x.canonical {
				t.Fatalf("incorrect canonical endpoint:\n- %s\n+ %s", x.canonical, end.String())
			}

			if len(end.Folders) != len(x.folders) {
				t.Fatalf("incorrect number of folders:\n- %#v\n+ %#v", x.folders, end.Folders)
			}

			for i, s := range end.Folders {
				if s != x.folders[i] {
					t.Fatalf("incorrect endpoint folder:\n- [%d] %#v\n+ [%d] %#v", i, x.folders, i, end.Folders)
				}
			}
		})
	}
}
