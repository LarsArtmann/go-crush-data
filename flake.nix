{
  description = "go-crush-data — typed, read-only Go access to Crush local session data";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    systems.url = "github:nix-systems/default";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;
        in
        {
          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs)
                go_1_26
                golangci-lint
                govulncheck
                golines
                nixfmt
                actionlint
                ;
            };

            GOTOOLCHAIN = "local";
          };

          packages.default = pkgs.runCommand "go-crush-data"
            {
              meta = with lib; {
                description = "Typed, read-only Go access to Crush local session data";
                homepage = "https://github.com/LarsArtmann/go-crush-data";
                license = licenses.mit;
                platforms = platforms.unix;
              };
            }
            ''
              ${goPkg}/bin/go build ./...
              mkdir -p $out
            '';

          apps = {
            test = {
              type = "app";
              program = toString (
                pkgs.writeShellApplication {
                  name = "test";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export CGO_ENABLED=0 GOTOOLCHAIN=local
                    exec go test -race "$@"
                  '';
                }
              );
            };

            lint = {
              type = "app";
              program = toString (
                pkgs.writeShellApplication {
                  name = "lint";
                  runtimeInputs = [
                    goPkg
                    pkgs.golangci-lint
                  ];
                  text = ''
                    export GOTOOLCHAIN=local
                    exec golangci-lint run ./...
                  '';
                }
              );
            };
          };

          treefmt = {
            programs = {
              nixfmt.enable = true;
              gofmt.enable = true;
            };
          };

          checks.build = config.packages.default;
          checks.format = config.treefmt.build.check self;
        };
    };
}
