{
  description = "go-crush-data — typed, read-only Go access to Crush local session data";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

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
      # Inline systems: nixpkgs 26.11 dropped x86_64-darwin, which github:nix-systems/default still lists.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          # go.mod's floor is 1.27 while nixpkgs' default go is 1.26.8, so the
          # toolchain has to be pinned here — for the package build (and its
          # go-modules layer) as much as for the devShell and the apps, because
          # the sandbox runs with GOTOOLCHAIN=local and will not download one.
          goPkg = pkgs.go_1_27;
        in
        {
          devShells.default = pkgs.mkShellNoCC {
            packages = builtins.attrValues {
              inherit (pkgs)
                go_1_27
                golangci-lint
                govulncheck
                golines
                nixfmt
                actionlint
                shellcheck
                ;
            };

            GOTOOLCHAIN = "local";
          };

          packages.default = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
            pname = "go-crush-data";
            version = self.rev or self.dirtyRev or "dev";
            src = ./.;
            vendorHash = "sha256-195155AMjIJzhEfZW8GLF+yAV+nIoQmEHxV1+0TAmxI=";

            meta = with lib; {
              description = "Typed, read-only Go access to Crush local session data";
              homepage = "https://github.com/LarsArtmann/go-crush-data";
              license = licenses.mit;
              platforms = platforms.unix;
            };

            env.CGO_ENABLED = "0";
            env.GOWORK = "off";
            doCheck = false;
          };

          apps = {
            test = {
              type = "app";
              meta.description = "Run the Go test suite with the race detector";
              program = "${
                pkgs.writeShellApplication {
                  name = "test";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export CGO_ENABLED=0 GOTOOLCHAIN=local
                    exec go test -race "$@" ./...
                  '';
                }
              }/bin/test";
            };

            lint = {
              type = "app";
              meta.description = "Run golangci-lint (args land before ./…)";
              program = "${
                pkgs.writeShellApplication {
                  name = "lint";
                  runtimeInputs = [
                    goPkg
                    pkgs.golangci-lint
                  ];
                  text = ''
                    export GOTOOLCHAIN=local
                    exec golangci-lint run "$@" ./...
                  '';
                }
              }/bin/lint";
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
