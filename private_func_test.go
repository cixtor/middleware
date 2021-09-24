package middleware

import (
	"path"
	"strings"
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

type InputEndpointMatch struct {
	name     string
	expected bool
	endpoint string
	rawURL   string
}

func TestEndpointMatch(t *testing.T) {
	inputs := []InputEndpointMatch{
		{name: "test_0", expected: true, endpoint: "/", rawURL: "/"},
		{name: "test_1", expected: true, endpoint: "/", rawURL: "////"},
		{name: "test_2", expected: true, endpoint: "/usr", rawURL: "/usr"},
		{name: "test_3", expected: true, endpoint: "/usr", rawURL: "/usr/"},
		{name: "test_4", expected: true, endpoint: "/usr", rawURL: "////usr"},
		{name: "test_5", expected: true, endpoint: "/usr", rawURL: "////usr////"},
		{name: "test_6", expected: false, endpoint: "/usr", rawURL: "/vsr"},
		{name: "test_7", expected: true, endpoint: "/usr/local", rawURL: "/usr/local"},
		{name: "test_8", expected: false, endpoint: "/usr/local", rawURL: "/usr/lacol"},
		{name: "test_9", expected: false, endpoint: "/usr/local", rawURL: "/usr"},
		{name: "test_10", expected: true, endpoint: "/usr/:local", rawURL: "/usr/local"},
		{name: "test_11", expected: true, endpoint: "/usr/:group", rawURL: "/usr/local"},
		{name: "test_12", expected: false, endpoint: "/usr/:group", rawURL: "/usr/local/etc"},
		{name: "test_13", expected: true, endpoint: "/usr/local/etc/openssl/cert.pem", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_14", expected: true, endpoint: "/usr/local/etc/openssl/:filename", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_15", expected: true, endpoint: "/usr/local/etc/:package/:filename", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_16", expected: true, endpoint: "/usr/local/:group/:package/:filename", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_17", expected: true, endpoint: "/usr/local/:group/:package/*", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_18", expected: true, endpoint: "/usr/local/:group/*", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_19", expected: true, endpoint: "/usr/local/*", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_20", expected: true, endpoint: "/usr/*", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_21", expected: true, endpoint: "/*", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_22", expected: true, endpoint: "/usr/local/:group/:package/cert.pem", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_23", expected: true, endpoint: "/usr/local/:group/openssl/cert.pem", rawURL: "/usr/local/etc/openssl/cert.pem"},
		{name: "test_24", expected: false, endpoint: "/usr/local/:group/openssl/cert.pem", rawURL: "/usr/local/etc/openzzl/cert.pem"},
		{name: "test_25", expected: false, endpoint: "/usr/local/:group/:package/cert.pem", rawURL: "/usr/local/etc/openssl/cert.key"},
		{name: "test_26", expected: false, endpoint: "/usr/local/:group/:package/*", rawURL: "/usr/local/etc/openssl"},
	}

	for _, x := range inputs {
		t.Run(x.name, func(t *testing.T) {
			arr := strings.Split(path.Clean(x.rawURL), "/")
			end := parseEndpoint(x.endpoint, nil)

			if _, ok := end.Match(arr); ok != x.expected {
				t.Fatalf("incorrect endpoint match result:\n- %s\n~ %s\n+ %s", x.endpoint, x.rawURL, end.String())
			}
		})
	}
}
