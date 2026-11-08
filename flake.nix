{
  description = "browser-selector";

  inputs = {
    nixpkgs = {
      url = "https://channels.nixos.org/nixos-unstable/nixexprs.tar.zst";
    };
  };

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      packages.${system}.default = pkgs.callPackage ./default.nix { };

      devShells.${system}.default = pkgs.mkShell {
        nativeBuildInputs = with pkgs; [
          go
          pkg-config
          wrapGAppsHook4
        ];
        buildInputs = with pkgs; [
          gtk4
          glib
          gobject-introspection
        ];
      };
    };
}
