{
  description = "Multi-agent orchestration system for Claude Code with persistent work tracking";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    beads = {
      url = "github:slwst/beads/chore/v0.63.3-flake-build";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      beads
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        beads = self.inputs.beads.packages.${system};
      in
      {
        packages = {
          gt = pkgs.buildGoLatestModule rec {
            pname = "gt";
            version = "0.13.0";
            src = ./.;
            vendorHash = "sha256-A/BqG1/CXBLVJdICgTbcP2GO1M/MZfcCTWU3cYVbx9M=";

            checkFlags = [ "-skip=^TestCrossPlatformBuild$" ];

            ldflags = [
              "-X github.com/steveyegge/gastown/internal/cmd.Version=${version}"
              "-X github.com/steveyegge/gastown/internal/cmd.Build=nix"
              "-X github.com/steveyegge/gastown/internal/cmd.BuiltProperly=1"
            ];

            subPackages = [ "cmd/gt" ];

            meta = with pkgs.lib; {
              description = "Multi-agent orchestration system for Claude Code with persistent work tracking";
              homepage = "https://github.com/steveyegge/gastown";
              license = licenses.mit;
              mainProgram = "gt";
            };
          };
          default = self.packages.${system}.gt;
        };

        apps = {
          gt = flake-utils.lib.mkApp {
            drv = self.packages.${system}.gt;
          };
          default = self.apps.${system}.gt;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            beads
            go
            gopls
            gotools
            go-tools
          ];
        };
      }
    );
}
