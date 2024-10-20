{
  description = "Find upcoming NC DMV appointments";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, ... }: let
    # No chromium on darwin: https://github.com/NixOS/nixpkgs/issues/247855
    supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    pname = "ncdmv";
    owner = "aksiksi";
    version = "0.1.0";
  in {
    packages = forAllSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        selfPackages = self.outputs.packages.${system};
      in {
        default = pkgs.buildGoModule {
          inherit pname;
          inherit version;
          src = ./.;
          vendorHash = "sha256-IhB0urJVKYbIIFSoiuR92/KlTe+gxOcIuLEZrZYyxO0=";
          buildInputs = [
            pkgs.sqlite
          ];
        };
        # Base image for Chromium to ensure that ncdmv gets its own layer in
        # the final image.
        #
        # If we put everything in one image, Nix will include Chromium and
        # ncdmv in the same (final) layer.
        chromium = pkgs.dockerTools.buildLayeredImage {
          name = "chromium-base";
          contents = [
            # Required for Discord webhook over HTTPS.
            pkgs.cacert
            pkgs.chromium
          ];
          extraCommands = ''
            # Required for Chromium to run
            mkdir tmp
          '';
          # Default is 100, so this ensures the final image gets its own
          # layer(s) after being merged with this base image.
          maxLayers = 90;
        };
        docker = pkgs.dockerTools.streamLayeredImage {
          name = "ghcr.io/aksiksi/ncdmv";
          tag = "latest";
          fromImage = selfPackages.chromium;
          # Use commit date to ensure image creation date is reproducible.
          created = builtins.substring 0 8 self.lastModifiedDate;
          config = {
            Entrypoint = [ "${selfPackages.default}/bin/ncdmv" ];
            Volumes = {
              # DB storage
              "/config" = null;
            };
            Env = [
              "NCDMV_APPT_TYPE="
              "NCDMV_DATABASE_PATH=/config/ncdmv.db"
              "NCDMV_LOCATIONS="
              "NCDMV_DISCORD_WEBHOOK="
              "NCDMV_TIMEOUT=5m0s"
              "NCDMV_INTERNVAL=5m0s"
            ];
          };
        };
      }
    );

    # Development shell
    # nix develop
    devShells = forAllSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        default = pkgs.mkShell {
          buildInputs = [
            pkgs.chromium
            pkgs.sqlite
          ];
          packages = [
            pkgs.go
            pkgs.gopls
          ];
        };
      }
    );
  };
}

