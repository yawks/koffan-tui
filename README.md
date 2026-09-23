# koffan-tui

Terminal interface for the [Koffan REST API](https://github.com/PanSalut/Koffan/wiki/REST-API).

This project was vibe-coded.

## Run

```sh
go run .
```

On first launch, the application creates `koffan-tui/config.json` in the user configuration directory. Then add your token:

```json
{
  "base_url": "http://localhost:3000",
  "token": "your-token"
}
```

The file is created with `0600` permissions.

## Keyboard shortcuts

- `Tab`, `←`, `→`, `↑`, `↓`: navigate
- `←` / `→` on a section: collapse/expand
- `Enter`: open a list or expand a section
- `Space` or `Enter`: complete/reopen an item
- `e`: edit the selected list, section, or item
- `n`, `s`, `a`: add a list, section, or item
- `Backspace` or `Delete`: delete after confirmation
- `r`: refresh
- `u`: hide/show the completed items column
- `c`: collapse/expand all sections
- `q`: quit
