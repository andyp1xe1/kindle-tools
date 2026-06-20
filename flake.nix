{
  description = "Kindle scriptlets and tools";

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
        echo "  kindle-tools dev shell"
        echo "  run dev:    make dev-<tool>"
        echo "  build arm:  make build-<tool>"
        echo "  package:    make package-<tool>"
        echo "  install:    make install"
        echo "  lint/fmt:   make lint  |  make fmt"
      '';
    };
  };
}
