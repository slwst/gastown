{
  description = "Multi-agent orchestration system for Claude Code with persistent work tracking";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    beads = {
      url = "github:steveyegge/beads/v0.55.4";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, beads }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        beads = self.inputs.beads.packages.${system};
      in
      {
        packages = {
          gt = pkgs.buildGoModule {
            pname = "gt";
            version = "0.8.0";
            src = ./.;
            vendorHash = "sha256-N1gMI9gflD6CKmo/RiuBzEeCD+0bUAGSrbm8qaGwR0E=";
            buildInputs = [ pkgs.icu ];
            env.GOTOOLCHAIN = "auto";
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
