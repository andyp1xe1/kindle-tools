{
  description = "Wallpaper manager for Kindle (KUAL applet + web UI)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {nixpkgs, ...}: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = [
        pkgs.go
        pkgs.zip
        pkgs.shellcheck
        pkgs.alejandra
        pkgs.biome
      ];

      shellHook = ''
        echo "  wallpapers dev shell"
        echo "  run dev:    make dev"
        echo "  build arm:  make build"
        echo "  package:    make package"
        echo "  lint/fmt:   make lint  |  make fmt"
      '';
    };
  };
}
