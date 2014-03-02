// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

(function() {
  function playground(opts) {
    var fillData = null;
    var fillRequest = null;
    function fill(term, callback) {
      if (fillRequest) {
        fillRequest.abort();
      }
      var code = $(opts.codeEl)
      var data = {
        "src": code[0].value,
        "offset": code[0].selectionStart,
        "filename": "fill.go"
      };
      window.console.log
      fillRequest = $.ajax("/fill", {
        data: data,
        type: "POST",
        dataType: "json",
        success: function(data) {
          fillRequest = null;
          if (data.Error) {
            window.console.log("fill error: ");
            window.console.log(data.Error);
          } else {
            window.console.log("fill data:");
            window.console.log(data);
            fillData = data;
            var res = [];
            if (data.Suggest) {
              for (var i = 0; i < data.Suggest.length; i++) {
                res.push(data.Suggest[i].Name);
              }
            }
            callback(res);
          }
        },
        error: function() { callback([]); }
      })
    }

    //$(opts.runEl).click(run);
    //$(opts.fmtEl).click(fmt);
    //$(opts.codeEl).bind('input propertychange', function() { window.console.log("propertychange"); });

    window.console.log("setting up autocomplete");
    $(opts.codeEl).textcomplete([{
      match: /(\w)\.(\w*)$/,
      search: fill,
      replace: function(value) {
        return "$1." + value;
      },
     }]).on({
      'textComplete:select': function(e, value) {
        window.console.log(value);
      },
      'textComplete:show': function(e) {
        window.console.log(e);
      },
      'textComplete:hide': function(e) {
        $("#doc").html("");
      },
      'textComplete:activate': function(e, value) {
        var doc = "";
        var index = parseInt(value.attributes["data-index"].value, 10);
        var suggest = fillData.Suggest[index];
        if (suggest) {
          doc = suggest.Doc;
        }
        $("#doc").html(doc);
      },
     });
  }

  window.playground = playground;
})();
