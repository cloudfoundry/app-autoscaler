sap.ui.define([
  "sap/m/App",
  "sap/m/Page",
  "sap/m/VBox",
  "sap/m/HBox",
  "sap/m/Title",
  "sap/m/Text",
  "sap/m/Button",
  "sap/m/ObjectStatus",
  "sap/m/OverflowToolbar",
  "sap/m/ToolbarSpacer",
  "sap/ui/core/Icon"
], function (App, Page, VBox, HBox, Title, Text, Button, ObjectStatus, OverflowToolbar, ToolbarSpacer, Icon) {
  "use strict";

  function feature(number, icon, title, description) {
    return new VBox({
      items: [
        new HBox({
          alignItems: "Center",
          justifyContent: "SpaceBetween",
          items: [
            new Text({ text: number }).addStyleClass("featureNumber"),
            new Icon({ src: icon, decorative: true }).addStyleClass("featureIcon")
          ]
        }),
        new Title({ text: title, level: "H2" }).addStyleClass("featureTitle"),
        new Text({ text: description }).addStyleClass("featureDescription")
      ]
    }).addStyleClass("featureCard");
  }

  var header = new OverflowToolbar({
    content: [
      new Text({ text: "AI" }).addStyleClass("campMark"),
      new Title({ text: "Code Camp", level: "H2" }).addStyleClass("campBrand"),
      new ToolbarSpacer(),
      new ObjectStatus({ text: "SAPUI5 · Cloud Foundry", icon: "sap-icon://cloud" }).addStyleClass("headerBadge")
    ]
  }).addStyleClass("campHeader");

  var hero = new VBox({
    alignItems: "Center",
    items: [
      new Text({ text: "LESS IDLE COMPUTE. MORE POSSIBILITY." }).addStyleClass("eyebrow"),
      new HBox({
        alignItems: "Center",
        justifyContent: "Center",
        items: [new Icon({ src: "sap-icon://cloud", decorative: true }).addStyleClass("heroCloud")]
      }).addStyleClass("cloudHalo"),
      new Title({ text: "Hello, AI Code Camp!", level: "H1", wrapping: true }).addStyleClass("heroTitle"),
      new Text({ text: "A tiny page. A big idea." }).addStyleClass("heroLead"),
      new Text({
        text: "This app can sleep at zero instances. One request brings it back to life.",
        textAlign: "Center"
      }).addStyleClass("heroDescription"),
      new Button({
        text: "Say hello again",
        icon: "sap-icon://refresh",
        type: "Emphasized",
        press: function () {
          // A new URL avoids a cached navigation when demonstrating another wake.
          // There are deliberately no timers or health polls keeping the app awake.
          var url = new URL(window.location.href);
          url.searchParams.set("hello", Date.now().toString());
          window.location.assign(url.href);
        }
      }).addStyleClass("helloButton"),
      new Text({ text: "Let it rest. Reload when you're ready to wake it." }).addStyleClass("heroHint")
    ]
  }).addStyleClass("hero");

  var features = new HBox({
    wrap: "Wrap",
    items: [
      feature("01 / REST", "sap-icon://history", "Sleep at zero", "No app instances while idle. Save the compute for when it matters."),
      feature("02 / REQUEST", "sap-icon://lightbulb", "Wake on demand", "The activator holds your first request while the app starts."),
      feature("03 / HELLO", "sap-icon://world", "Keep the same URL", "The app comes back to life and serves this page. No URL changes, no manual scaling.")
    ]
  }).addStyleClass("features");

  var content = new VBox({
    items: [
      hero,
      features,
      new Text({ text: "Built for curious minds. Powered by SAPUI5 and Cloud Foundry." }).addStyleClass("campFooter")
    ]
  }).addStyleClass("campContent");

  var page = new Page({
    customHeader: header,
    backgroundDesign: "Transparent",
    content: [content]
  }).addStyleClass("campPage");

  new App({ pages: [page] }).placeAt("ui5-root");
  document.getElementById("fallback").hidden = true;
  document.getElementById("ui5-root").setAttribute("aria-busy", "false");
});
