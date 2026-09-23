# koffan-tui

Interface terminal pour l'[API REST de Koffan](https://github.com/PanSalut/Koffan/wiki/REST-API).

## Lancer

```sh
go run .
```

Au premier lancement, l'application crée `koffan-tui/config.json` dans le dossier de configuration utilisateur. Renseignez ensuite le token :

```json
{
  "base_url": "http://localhost:3000",
  "token": "votre-token"
}
```

Le fichier est créé avec des permissions `0600`.

## Raccourcis

- `Tab`, `←`, `→`, `↑`, `↓` : navigation
- `←` / `→` sur une section : replier/déplier
- `Entrée` : ouvrir une liste ou déplier une section
- `Espace` ou `Entrée` : terminer/réouvrir un article
- `e` : modifier la liste, la section ou l'article sélectionné
- `n`, `s`, `a` : ajouter une liste, une section ou un article
- `Retour arrière` ou `Suppr` : supprimer après confirmation
- `r` : rafraîchir
- `u` : masquer/afficher la colonne des articles terminés
- `c` : replier/déplier toutes les sections
- `q` : quitter
