{
  description = "A basic gomod2nix flake";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.gomod2nix.url = "github:nix-community/gomod2nix";

  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    (flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ gomod2nix.overlays.default ];
          };

          everything = pkgs.buildGoApplication {
            pname = "go-home";
            version = "1.0.0";
            src = ./.;
            modules = ./gomod2nix.toml;

            buildInputs = with pkgs; [ pkg-config libaom libavif ];
          };
        in
        {
          packages.default = everything;
          devShells.default = import ./shell.nix { inherit pkgs; };
        })
    );
}
