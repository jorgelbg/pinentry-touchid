{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
      go
      gopls
      goimports
      darwin.apple_sdk.frameworks.CoreFoundation
      darwin.apple_sdk.frameworks.Foundation
      darwin.apple_sdk.frameworks.LocalAuthentication

      # keep this line if you use bash
      pkgs.bashInteractive
    ];

  shellHook = ''
    unset GOPATH GOROOT
    export NIX_LDFLAGS="-F${pkgs.darwin.apple_sdk.frameworks.CoreFoundation}/Library/Frameworks -framework CoreFoundation $NIX_LDFLAGS";
  '';
}
