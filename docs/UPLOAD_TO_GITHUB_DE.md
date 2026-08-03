# Upload zu GitHub

1. Dieses Repository-ZIP vollständig in einen leeren Ordner entpacken.
2. Auf GitHub ein leeres Repository ohne automatisch erzeugtes README anlegen.
3. Im entpackten Ordner ausführen:

```bash
git init
git add -A
git commit -m "ECCO2 CPWI Dew Mirror 1.0.1"
git branch -M main
git remote add origin https://github.com/DEIN-NAME/DEIN-REPOSITORY.git
git push -u origin main
```

Der Workflow startet automatisch. Nach erfolgreichem Lauf stehen MSI, Setup-EXE, portable ZIP, Source-ZIP und Prüfsummen unter **Actions → Artifacts** bereit.

Für ein Release: **Actions → Build, test and package → Run workflow**, `publish_release=true` und `release_tag=v1.0.1`.
