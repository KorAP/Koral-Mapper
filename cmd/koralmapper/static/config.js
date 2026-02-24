"use strict";

(function () {
  var container = document.querySelector(".container");
  if (!container) return;

  var serviceURL = container.dataset.serviceUrl;
  var cookieName = container.dataset.cookieName || "km-config";
  var requestMappingDivs = container.querySelectorAll('.mapping[data-mode="request"]');
  var responseMappingDivMap = {};
  var requestCfgPreview = container.querySelector(".request-cfg-preview");
  var responseCfgPreview = container.querySelector(".response-cfg-preview");

  var responseMappingDivs = container.querySelectorAll('.mapping[data-mode="response"]');
  for (var i = 0; i < responseMappingDivs.length; i++) {
    responseMappingDivMap[responseMappingDivs[i].dataset.id] = responseMappingDivs[i];
  }

  // Cookie helpers

  function readCookie() {
    var prefix = cookieName + "=";
    var parts = document.cookie.split("; ");
    for (var i = 0; i < parts.length; i++) {
      if (parts[i].indexOf(prefix) === 0) {
        try {
          return JSON.parse(decodeURIComponent(parts[i].substring(prefix.length)));
        } catch (e) {
          return null;
        }
      }
    }
    return null;
  }

  function writeCookie(state) {
    var value = encodeURIComponent(JSON.stringify(state));
    document.cookie = cookieName + "=" + value + "; path=/; SameSite=Lax; max-age=31536000";
  }

  function deleteCookie() {
    document.cookie = cookieName + "=; path=/; SameSite=Lax; max-age=0";
  }

  // Form state

  function reverseDir(dir) {
    return dir === "atob" ? "btoa" : "atob";
  }

  function rowFieldClasses(mode) {
    return {
      foundryA: "." + mode + "-foundryA",
      layerA: "." + mode + "-layerA",
      foundryB: "." + mode + "-foundryB",
      layerB: "." + mode + "-layerB",
      fieldA: "." + mode + "-fieldA",
      fieldB: "." + mode + "-fieldB",
      dirArrow: "." + mode + "-dir-arrow"
    };
  }

  function inputValue(parent, selector) {
    var el = parent.querySelector(selector);
    return el ? el.value : "";
  }

  function getModeState(div, mode) {
    var classes = rowFieldClasses(mode);
    var arrow = div.querySelector(classes.dirArrow);
    return {
      enabled: div.querySelector("." + mode + "-cb").checked,
      dir: arrow ? arrow.dataset.dir : "atob",
      foundryA: inputValue(div, classes.foundryA),
      layerA: inputValue(div, classes.layerA),
      foundryB: inputValue(div, classes.foundryB),
      layerB: inputValue(div, classes.layerB),
      fieldA: inputValue(div, classes.fieldA),
      fieldB: inputValue(div, classes.fieldB)
    };
  }

  function getFormState() {
    var state = { mappings: [] };

    for (var i = 0; i < requestMappingDivs.length; i++) {
      var requestDiv = requestMappingDivs[i];
      var responseDiv = responseMappingDivMap[requestDiv.dataset.id];
      var entry = {
        id: requestDiv.dataset.id
      };

      entry.request = getModeState(requestDiv, "request");
      entry.response = responseDiv ? getModeState(responseDiv, "response") : { enabled: false };

      state.mappings.push(entry);
    }

    return state;
  }

  // Restore form from cookie

  function setInputValue(parent, selector, value) {
    if (value === undefined) return;
    var el = parent.querySelector(selector);
    if (el) el.value = value;
  }

  function setArrowDirection(div, selector, dir) {
    var arrow = div.querySelector(selector);
    if (!arrow || !dir) return;
    arrow.dataset.dir = dir;
    arrow.textContent = dir === "atob" ? "\u2192" : "\u2190";
  }

  function restoreModeState(div, mode, modeState) {
    if (!modeState) return;
    var classes = rowFieldClasses(mode);
    var checkbox = div.querySelector("." + mode + "-cb");
    if (checkbox) checkbox.checked = !!modeState.enabled;
    setArrowDirection(div, classes.dirArrow, modeState.dir);
    setInputValue(div, classes.foundryA, modeState.foundryA);
    setInputValue(div, classes.layerA, modeState.layerA);
    setInputValue(div, classes.foundryB, modeState.foundryB);
    setInputValue(div, classes.layerB, modeState.layerB);
    setInputValue(div, classes.fieldA, modeState.fieldA);
    setInputValue(div, classes.fieldB, modeState.fieldB);
  }

  function restoreFormState(saved) {
    if (!saved || !saved.mappings) return;

    var byId = {};
    for (var i = 0; i < saved.mappings.length; i++) {
      byId[saved.mappings[i].id] = saved.mappings[i];
    }

    for (var i = 0; i < requestMappingDivs.length; i++) {
      var requestDiv = requestMappingDivs[i];
      var responseDiv = responseMappingDivMap[requestDiv.dataset.id];
      var entry = byId[requestDiv.dataset.id];
      if (!entry) continue;

      if (requestDiv.dataset.type !== "corpus") {
        // Backward compatibility with old cookie schema.
        if (entry.request && typeof entry.request === "object") {
          restoreModeState(requestDiv, "request", entry.request);
          if (responseDiv) {
            restoreModeState(responseDiv, "response", entry.response);
          }
        } else {
          var requestLegacy = {
            enabled: !!entry.request,
            dir: entry.dir || "atob",
            foundryA: entry.foundryA,
            layerA: entry.layerA,
            foundryB: entry.foundryB,
            layerB: entry.layerB
          };
          var responseLegacy = {
            enabled: !!entry.response,
            dir: reverseDir(entry.dir || "atob"),
            foundryA: entry.foundryA,
            layerA: entry.layerA,
            foundryB: entry.foundryB,
            layerB: entry.layerB
          };
          restoreModeState(requestDiv, "request", requestLegacy);
          if (responseDiv) {
            restoreModeState(responseDiv, "response", responseLegacy);
          }
        }
      } else {
        // Backward compatibility with old cookie schema.
        if (entry.request && typeof entry.request === "object") {
          restoreModeState(requestDiv, "request", entry.request);
          if (responseDiv) {
            restoreModeState(responseDiv, "response", entry.response);
          }
        } else {
          var requestCb = requestDiv.querySelector(".request-cb");
          var responseCb = responseDiv ? responseDiv.querySelector(".response-cb") : null;
          if (requestCb) {
            requestCb.checked = !!entry.request;
          }
          if (responseCb) {
            responseCb.checked = !!entry.response;
          }
        }
      }
    }
  }

  // cfg parameter building

  // Returns "" when the input matches its default (compact URL).
  function cfgFieldValue(div, inputSelector, defaultDataAttr) {
    var el = div.querySelector(inputSelector);
    if (!el) return "";
    var val = el.value;
    var def = div.dataset[defaultDataAttr] || "";
    return val === def ? "" : val;
  }

  function buildCfgParam(mode) {
    var parts = [];
    var classes = rowFieldClasses(mode);
    var mappingDivs = mode === "request" ? requestMappingDivs : responseMappingDivs;

    for (var i = 0; i < mappingDivs.length; i++) {
      var div = mappingDivs[i];
      var cbClass = mode === "request" ? ".request-cb" : ".response-cb";
      var cb = div.querySelector(cbClass);
      if (!cb || !cb.checked) continue;

      var id = div.dataset.id;
      var dir = "atob";
      var arrow = div.querySelector(classes.dirArrow);
      dir = arrow ? arrow.dataset.dir : "atob";

      if (div.dataset.type !== "corpus") {
        var fA = cfgFieldValue(div, classes.foundryA, "defaultFoundryA");
        var lA = cfgFieldValue(div, classes.layerA, "defaultLayerA");
        var fB = cfgFieldValue(div, classes.foundryB, "defaultFoundryB");
        var lB = cfgFieldValue(div, classes.layerB, "defaultLayerB");

        if (fA || lA || fB || lB) {
          parts.push(id + ":" + dir + ":" + fA + ":" + lA + ":" + fB + ":" + lB);
        } else {
          parts.push(id + ":" + dir);
        }
      } else {
        var fieldA = cfgFieldValue(div, classes.fieldA, "defaultFieldA");
        var fieldB = cfgFieldValue(div, classes.fieldB, "defaultFieldB");
        if (fieldA || fieldB) {
          parts.push(id + ":" + dir + ":" + fieldA + ":" + fieldB);
        } else {
          parts.push(id + ":" + dir);
        }
      }
    }

    return parts.join(";");
  }

  // Kalamar pipe registration

  var lastQueryPipe = null;
  var lastResponsePipe = null;

  function registerPipes() {
    var queryCfg = buildCfgParam("request");
    var responseCfg = buildCfgParam("response");

    if (requestCfgPreview) {
      requestCfgPreview.value = queryCfg;
    }
    if (responseCfgPreview) {
      responseCfgPreview.value = responseCfg;
    }

    var newQueryPipe = queryCfg ? serviceURL + "/query?cfg=" + encodeURIComponent(queryCfg) : "";
    var newResponsePipe = responseCfg ? serviceURL + "/response?cfg=" + encodeURIComponent(responseCfg) : "";

    if (newQueryPipe === lastQueryPipe && newResponsePipe === lastResponsePipe) return;

    if (typeof KorAPlugin !== "undefined") {
      if (newQueryPipe !== lastQueryPipe) {
        if (lastQueryPipe) {
          KorAPlugin.sendMsg({ action: "pipe", job: "del", service: lastQueryPipe });
        }
        if (newQueryPipe) {
          KorAPlugin.sendMsg({ action: "pipe", job: "add", service: newQueryPipe });
        }
      }
      if (newResponsePipe !== lastResponsePipe) {
        if (lastResponsePipe) {
          KorAPlugin.sendMsg({ action: "pipe", job: "del-after", service: lastResponsePipe });
        }
        if (newResponsePipe) {
          KorAPlugin.sendMsg({ action: "pipe", job: "add-after", service: newResponsePipe });
        }
      }
    }

    lastQueryPipe = newQueryPipe;
    lastResponsePipe = newResponsePipe;
  }

  // Change handler

  function onChange() {
    writeCookie(getFormState());
    registerPipes();
  }

  // Initialisation

  var saved = readCookie();
  if (saved) {
    restoreFormState(saved);
  }

  var checkboxes = container.querySelectorAll('input[type="checkbox"]');
  for (var i = 0; i < checkboxes.length; i++) {
    checkboxes[i].addEventListener("change", onChange);
  }

  var textInputs = container.querySelectorAll('input[type="text"]');
  for (var i = 0; i < textInputs.length; i++) {
    textInputs[i].addEventListener("input", onChange);
  }

  var arrows = container.querySelectorAll(".request-dir-arrow, .response-dir-arrow");
  for (var i = 0; i < arrows.length; i++) {
    (function (arrow) {
      arrow.addEventListener("click", function () {
        var next = reverseDir(arrow.dataset.dir);
        arrow.dataset.dir = next;
        arrow.textContent = next === "atob" ? "\u2192" : "\u2190";
        onChange();
      });
    })(arrows[i]);
  }

  // Reset button

  function resetForm() {
    for (var i = 0; i < checkboxes.length; i++) {
      checkboxes[i].checked = false;
    }

    for (var i = 0; i < textInputs.length; i++) {
      textInputs[i].value = "";
    }

    var requestArrows = container.querySelectorAll(".request-dir-arrow");
    for (var i = 0; i < requestArrows.length; i++) {
      requestArrows[i].dataset.dir = "atob";
      requestArrows[i].textContent = "\u2192";
    }
    var responseArrows = container.querySelectorAll(".response-dir-arrow");
    for (var i = 0; i < responseArrows.length; i++) {
      responseArrows[i].dataset.dir = "btoa";
      responseArrows[i].textContent = "\u2190";
    }

    deleteCookie();
    registerPipes();
  }

  var resetBtn = container.querySelector("#reset-btn");
  if (resetBtn) {
    resetBtn.addEventListener("click", resetForm);
  }

  registerPipes();
})();
