// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func SimpleHandler() *Handler {
	return &Handler{ SimpleIndexer() }
}

type Handler struct {
	x *Index
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, editorHTML)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "GET or POST only", 500)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	//filename := r.PostFormValue("filename")
	src := r.PostFormValue("src")
	fmt.Printf("src: %q\n", src)
	offset, err := strconv.Atoi(r.PostFormValue("offset"))
	if err != nil {
		http.Error(w, fmt.Sprintf("pos: %v", err), 500)
		return
	}
	if offset >= len(src) {
		offset = len(src) - 1
	}

	res := h.x.Query(src, offset)

	b, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(b)
}

var editorHTML = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<link type="text/css" rel="stylesheet" href="/lib/godoc/style.css">
<script type="text/javascript">window.initFuncs = [];</script>
</head>
<body style="margin: 20px;">

<div id="learn">
<div class="input">

<div id="doc">
</div>

<textarea spellcheck="false" class="code">package main

import "fmt"

func main() {
	fmt
}</textarea>
</div>
<div class="output">
<pre>
</pre>
</div>
<div class="buttons">
<a class="run" href="#" title="Run this code [shift-enter]">Run</a>
</div>
</div>

<script type="text/javascript" src="/lib/godoc/jquery.js"></script>
<script type="text/javascript" src="/lib/godoc/jquery.textcomplete.js"></script>
<script type="text/javascript" src="/lib/godoc/playground.js"></script>

<script type="text/javascript">
	if (window.playground) {
		window.playground({
			"codeEl":        "#learn .code",
			"outputEl":      "#learn .output",
			"runEl":         "#learn .run",
		});
	}
</script>

</body>
</html>
`
