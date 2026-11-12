{
  description = "A client for the PLaMo 3 Translate for multi-language translation running on ollama";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, ... }:
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        {
          pkgs,
          config,
          self',
          inputs',
          lib,
          system,
          ...
        }:
        {
          packages = {
            default = self'.packages.bin;

            bin = pkgs.buildGoModule (finalAttrs: {
              pname = "plamo-client";
              version = self.shortRev or self.dirtyShortRev or "dev";
              src = ./.;
              vendorHash = "sha256-2fwZ9bVeYbK26eA4sGiIY0k6kc6jciCajP3MnhfaUK8=";

              env.CGO_ENABLED = "0";

              ldflags = [
                "-s"
                "-w"
              ];

              meta.mainProgram = "plamo-client";
            });
          };

          devShells = {
            default = pkgs.mkShell {
              packages = [
                pkgs.go
              ];
            };
          };
        };
    };
}
