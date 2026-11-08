{
  lib,
  buildGoModule,
  makeDesktopItem,
  copyDesktopItems,
  pkg-config,
  wrapGAppsHook4,
  gtk4,
  glib,
  gobject-introspection,
}:

# Untested: written without nixpkgs access. After `go mod tidy` creates go.sum,
# build once with the fake hash below and paste the real vendorHash from the error.
buildGoModule {
  pname = "browser-selector";
  version = "0.2.0";
  src = ./.;
  vendorHash = "sha256-+Y80+SEFv7T32OHsB4/zU7CnY04L4Ec649gUD2yaGmQ=";

  nativeBuildInputs = [
    pkg-config
    wrapGAppsHook4
    copyDesktopItems
  ];
  buildInputs = [
    gtk4
    glib
    gobject-introspection
  ];

  desktopItems = [
    (makeDesktopItem {
      name = "browser-selector";
      desktopName = "Browser Selector";
      exec = "browser-selector %u";
      categories = [
        "Network"
        "WebBrowser"
      ];
      mimeTypes = [
        "x-scheme-handler/http"
        "x-scheme-handler/https"
        "text/html"
      ];
    })
  ];

  meta = {
    mainProgram = "browser-selector";
  };
}
