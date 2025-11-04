{
  description = "mcp-grafana";

  inputs = {
    nixpkgs.url = "nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go_1_25;
      in
      {
        devShell = pkgs.mkShellNoCC {
          name = "mcp-grafana";
          nativeBuildInputs = with pkgs; [
            go
            golangci-lint
          ];
          CGO_ENABLED = 0;
        };
      }
    );
}
