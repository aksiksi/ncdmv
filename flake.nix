{
  description = "Find upcoming NC DMV appointments";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, ... }: let
    supportedSystems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    pname = "ncdmv";
    owner = "aksiksi";
    version = "0.1.0";
  in {
    packages = forAllSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        selfPackages = self.outputs.packages.${system};
        entrypoint = pkgs.writeScript "docker-entrypoint.sh" ''
          #!${pkgs.stdenv.shell}
          exec ${selfPackages.default}/bin/ncdmv \
            -t "$NCDMV_APPT_TYPE" \
            -l "$NCDMV_LOCATIONS" \
            -d "$NCDMV_DATABASE_PATH" \
            -w "$NCDMV_DISCORD_WEBHOOK" \
            --timeout "$NCDMV_TIMEOUT" \
            --interval "$NCDMV_INTERVAL" \
            --notify-unavailable=$NCDMV_NOTIFY_UNAVAILABLE \
            --disable-gpu=$NCDMV_DISABLE_GPU \
            --debug=$NCDMV_DEBUG \
            --debug-chrome=$NCDMV_DEBUG_CHROME
        '';
        # https://hub.docker.com/r/chromedp/headless-shell
        chrome-headless-amd64 = pkgs.dockerTools.pullImage {
          imageName = "docker.io/chromedp/headless-shell";
          imageDigest = "sha256:e79572ba4c81533484f97ce9dc4253a51aa5f6b13593f5c3408af005cf7526fa";
          finalImageTag = "latest";
          sha256 = "sha256-Aqq6Mjod1OJUKswdNh2M+85O/XMhRCNWmyn7dm3oO7U=";
          os = "linux";
          arch = "amd64";
        };
        chrome-headless-arm64 = pkgs.dockerTools.pullImage {
          imageName = "docker.io/chromedp/headless-shell";
          imageDigest = "sha256:0848400a41ce64e77f240854c212b32a1f50d65f098413181a62cba03889d556";
          finalImageTag = "latest";
          sha256 = "sha256-IpLyO2g8Vkr2gdANQw5ZwCzC9FhWAFGZ6adXQEXEIsA=";
          os = "linux";
          arch = "arm64";
        };
        imageArch =
          if pkgs.lib.strings.hasPrefix "x86_64" system
          then "amd64"
          else "arm64";
      in {
        default = pkgs.buildGoModule {
          inherit pname;
          inherit version;
          src = ./.;
          vendorHash = "sha256-enRqcu0brw0J7NPktMosRrI4W/nRg+oHk6INjgMTSBc=";
          buildInputs = [
            pkgs.sqlite
          ];
        };
        docker = pkgs.dockerTools.streamLayeredImage {
          name = "ghcr.io/aksiksi/ncdmv";
          tag = "latest";
          fromImage =
            if imageArch == "amd64"
            then chrome-headless-amd64
            else chrome-headless-arm64;
          # Use commit date to ensure image creation date is reproducible.
          created = builtins.substring 0 8 self.lastModifiedDate;
          architecture = imageArch;
          config = {
            Entrypoint = [ entrypoint ];
            Volumes = {
              # DB storage
              "/config" = {};
            };
            Env = [
              "NCDMV_APPT_TYPE="
              "NCDMV_LOCATIONS="
              "NCDMV_DATABASE_PATH=/config/ncdmv.db"
              "NCDMV_DISCORD_WEBHOOK="
              "NCDMV_TIMEOUT=5m"
              "NCDMV_INTERVAL=5m"
              "NCDMV_DISABLE_GPU=false"
              "NCDMV_NOTIFY_UNAVAILABLE=true"
              "NCDMV_DEBUG=false"
              "NCDMV_DEBUG_CHROME=false"
            ];
          };
          # Default is 100, so this ensures this image gets its own layer(s)
          # after being merged with the base image.
          maxLayers = 120;
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

