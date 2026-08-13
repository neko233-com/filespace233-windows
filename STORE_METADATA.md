# Microsoft Store Submission Metadata

This document is the source of truth for the Partner Center listing.

## Listing

- Product name: Filespace
- Category: Productivity
- Short description: A fast, keyboard-first file workspace for Windows.
- Keywords: file manager, dual pane, tabs, keyboard, search, Windows, Everything
- Support URL: https://github.com/neko233-com/filespace233-windows/issues
- Privacy URL: https://github.com/neko233-com/filespace233-windows/blob/main/PRIVACY.md
- Age rating: Everyone / 3+

## Full description

Filespace is a native WinUI 3 file workspace for Windows power users. Browse folders asynchronously, work across tabs and dual panes, navigate with the keyboard, and search files with Ctrl+K. The built-in search engine works without a network connection and can optionally bridge a local Everything installation through es.exe.

Win+F activates Filespace when the optional Windows startup task is enabled. Win+E is left unchanged and continues to open Windows Explorer.

## Store submission checklist

- Reserve the product name in Partner Center.
- Associate the project with the reserved Store identity and replace the placeholder Identity Name and Publisher in `Package.appxmanifest`.
- Set a four-part version greater than the last submitted version.
- Build the `StoreUpload` configuration and upload the generated `.msixupload` file.
- Upload screenshots at the required Store resolutions and confirm the listing text and privacy URL.
- Complete age rating, declaration, accessibility, and package flight checks.
- Submit for certification and address any automated or manual review findings.

The repository can prepare and validate the package, but Microsoft certification and publication are completed by Partner Center after the publisher account and Store identity are associated.
