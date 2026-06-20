// Silent jsrepl executor. Loaded by the host page (the wallpapers widget,
// which runs inside a registered Mesquite app so KNF bridges are wired
// in). Long-polls /replin, evals jobs in the host page's global scope,
// posts results to /replout. No UI — drive everything from the laptop.
(function () {
  // Derive the jsrepl base URL from this script's own src so the snippet
  // is portable across IPs/ports.
  var BASE = "";
  var scripts = document.getElementsByTagName("script");
  for (var i = 0; i < scripts.length; i++) {
    var src = scripts[i].src || "";
    var m = src.match(/^(.*?:\/\/[^/]+)\/executor\.js/);
    if (m) {
      BASE = m[1];
      break;
    }
  }
  if (!BASE) return; // can't find ourselves; bail rather than hammer 127.0.0.1

  function safeStr(v) {
    if (v === undefined) return "undefined";
    if (v === null) return "null";
    try {
      return JSON.stringify(v);
    } catch (_) {}
    try {
      return String(v);
    } catch (_) {
      return "<unstringifiable>";
    }
  }

  function post(out, cb) {
    var p = new XMLHttpRequest();
    p.open("POST", BASE + "/replout", true);
    p.setRequestHeader("Content-Type", "application/json");
    p.onreadystatechange = function () {
      if (p.readyState === 4) cb();
    };
    p.onerror = function () {
      cb();
    };
    try {
      p.send(JSON.stringify(out));
    } catch (_) {
      cb();
    }
  }

  function tick() {
    var x = new XMLHttpRequest();
    x.open("GET", BASE + "/replin", true);
    x.onreadystatechange = function () {
      if (x.readyState !== 4) return;
      if (x.status === 200 && x.responseText) {
        var job, out;
        try {
          job = JSON.parse(x.responseText);
        } catch (_e) {
          setTimeout(tick, 500);
          return;
        }
        try {
          // biome-ignore lint/security/noGlobalEval: jsrepl exists to eval host-sent code
          var v = eval(job.code);
          out = { id: job.id, ok: true, value: safeStr(v) };
        } catch (e) {
          out = { id: job.id, ok: false, error: String((e && e.stack) || e) };
        }
        post(out, function () {
          setTimeout(tick, 0);
        });
      } else if (x.status === 204) {
        setTimeout(tick, 0);
      } else {
        // jsrepl probably not running — back off so we don't busy-loop
        setTimeout(tick, 3000);
      }
    };
    x.onerror = function () {
      setTimeout(tick, 3000);
    };
    try {
      x.send();
    } catch (_) {
      setTimeout(tick, 3000);
    }
  }

  tick();
})();
