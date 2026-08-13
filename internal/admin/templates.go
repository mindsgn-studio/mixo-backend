package admin

const adminPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.delete { background: #dc3545; }
        button.delete:hover { background: #c82333; }
        button.add-queue { background: #28a745; }
        button.add-queue:hover { background: #218838; }
        button.restore { background: #17a2b8; }
        button.restore:hover { background: #138496; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; flex-wrap: wrap; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .now-playing { background: #e3f2fd; border-left: 4px solid #2196f3; padding: 15px; margin-bottom: 20px; border-radius: 5px; }
        .now-playing h3 { margin: 0 0 10px 0; color: #1976d2; }
        .now-playing p { margin: 5px 0; color: #555; }
        .status { font-weight: bold; }
        .status.playing { color: #28a745; }
        .status.paused { color: #ffc107; }
        .duration { font-family: monospace; }
        .favourite-btn { background: none; border: none; cursor: pointer; font-size: 18px; padding: 0; }
        .favourite-btn.favourited { color: #ffc107; }
        .favourite-btn:not(.favourited) { color: #ccc; }
        .drag-handle { cursor: grab; padding: 4px 8px; font-size: 16px; }
        .drag-handle:active { cursor: grabbing; }
        .sortable-ghost { opacity: 0.4; background: #e3f2fd; }
        select { padding: 6px 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 13px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin" class="active">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Radio Dashboard</h1>
        <div id="message"></div>

        <div class="section">
            <h2>Playback Control</h2>
            <div id="now-playing" hx-get="/admin/now-playing" hx-trigger="load, every 5s">
                {{NOW_PLAYING}}
            </div>
        </div>

        <div class="section">
            <h2>Playback Queue</h2>
            <div id="queue-table" hx-get="/admin/queue" hx-trigger="load, refresh, every 10s">
                {{QUEUE}}
            </div>
        </div>
    </div>
</body>
</html>`

const libraryPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Music Library - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        input, select { padding: 8px 12px; border: 1px solid #ddd; border-radius: 5px; font-size: 14px; }
        input:focus, select:focus { outline: none; border-color: #667eea; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.delete { background: #dc3545; }
        button.add-queue { background: #28a745; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .duration { font-family: monospace; }
        .favourite-btn { background: none; border: none; cursor: pointer; font-size: 18px; padding: 0; }
        .favourite-btn.favourited { color: #ffc107; }
        .favourite-btn:not(.favourited) { color: #ccc; }
        .search-bar { display: flex; gap: 10px; margin-bottom: 20px; flex-wrap: wrap; }
        .search-bar input, .search-bar select { flex: 1; min-width: 150px; }
        select { padding: 8px 12px; border: 1px solid #ddd; border-radius: 5px; font-size: 13px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library" class="active">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Music Library</h1>
        <div id="message"></div>

        <div class="section">
            <h2>Search & Filter</h2>
            <div class="search-bar">
                <input type="text" id="search-input" name="q" placeholder="Search title, artist, album, genre..."
                    hx-get="/admin/library/songs"
                    hx-trigger="keyup changed delay:300ms"
                    hx-target="#library-table"
                    hx-include="[name='genre'],[name='sort']"
                    name="q">
                <select name="genre" hx-get="/admin/library/songs" hx-trigger="change" hx-target="#library-table" hx-include="[name='q'],[name='sort']">
                    <option value="">All Genres</option>
                </select>
                <select name="sort" hx-get="/admin/library/songs" hx-trigger="change" hx-target="#library-table" hx-include="[name='q'],[name='genre']">
                    <option value="">Sort by Artist</option>
                    <option value="title">Sort by Title</option>
                    <option value="album">Sort by Album</option>
                    <option value="genre">Sort by Genre</option>
                    <option value="duration">Sort by Duration</option>
                </select>
            </div>
            <div id="library-table">
                {{LIBRARY}}
            </div>
        </div>
    </div>
</body>
</html>`

const playlistsPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Playlists - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        input { padding: 8px 12px; border: 1px solid #ddd; border-radius: 5px; font-size: 14px; }
        input:focus { outline: none; border-color: #667eea; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.delete { background: #dc3545; }
        button.add-queue { background: #28a745; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .create-form { display: flex; gap: 10px; margin-bottom: 20px; }
        .create-form input { flex: 1; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists" class="active">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Playlists</h1>
        <div id="message"></div>

        <div class="section">
            <h2>Create Playlist</h2>
            <form class="create-form" hx-post="/admin/playlists/create" hx-target="#message" hx-swap="innerHTML" hx-on::after-request="if(event.detail.successful) { this.reset(); htmx.trigger('#playlists-table', 'refresh'); }">
                <input type="text" name="name" placeholder="Playlist name" required>
                <button type="submit">Create</button>
            </form>

            <h2>Your Playlists</h2>
            <div id="playlists-table" hx-get="/admin/playlists/list" hx-trigger="load, refresh">
                {{PLAYLISTS}}
            </div>
        </div>
    </div>
</body>
</html>`

const playlistDetailPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{PLAYLIST_NAME}} - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.delete { background: #dc3545; }
        button.add-queue { background: #28a745; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .duration { font-family: monospace; }
        .drag-handle { cursor: grab; padding: 4px 8px; font-size: 16px; }
        .sortable-ghost { opacity: 0.4; background: #e3f2fd; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists" class="active">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>{{PLAYLIST_NAME}}</h1>
        <div id="message"></div>

        <div class="section">
            <div style="display:flex;gap:10px;margin-bottom:20px;">
                <button class="add-queue" hx-post="/admin/playlists/{{PLAYLIST_ID}}/queue" hx-target="#message" hx-swap="innerHTML">Queue All</button>
                <a href="/admin/playlists" style="padding:8px 16px;color:#667eea;text-decoration:none;">Back to Playlists</a>
            </div>
            <div id="playlist-songs" hx-get="/admin/playlists/{{PLAYLIST_ID}}/songs" hx-trigger="load, refresh">
                {{SONGS}}
            </div>
        </div>
    </div>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            var el = document.getElementById('playlist-songs-tbody');
            if (el) {
                new Sortable(el, {
                    handle: '.drag-handle',
                    animation: 150,
                    ghostClass: 'sortable-ghost',
                    onEnd: function(evt) {
                        var songId = evt.item.dataset.songId;
                        var playlistId = evt.item.dataset.playlistId;
                        var newPosition = evt.newIndex + 1;
                        htmx.ajax('POST', '/admin/playlists/' + playlistId + '/reorder', {
                            values: { playlist_id: playlistId, song_id: songId, new_position: newPosition },
                            target: '#playlist-songs'
                        });
                    }
                });
            }
        });
    </script>
</body>
</html>`

const albumsPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Albums - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 20px; }
        .album-card { background: #f8f9fa; border-radius: 10px; padding: 20px; text-align: center; text-decoration: none; color: #333; transition: transform 0.2s, box-shadow 0.2s; }
        .album-card:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
        .album-name { font-weight: 600; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album" class="active">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Albums</h1>
        <div class="section">
            <div id="albums-grid" hx-get="/admin/library/album/list" hx-trigger="load">
                {{ALBUMS}}
            </div>
        </div>
    </div>
</body>
</html>`

const albumDetailPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ALBUM_NAME}} - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.add-queue { background: #28a745; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .duration { font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album" class="active">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>{{ALBUM_NAME}}</h1>
        <div id="message"></div>
        <div class="section">
            <a href="/admin/library/album" style="color:#667eea;text-decoration:none;margin-bottom:20px;display:inline-block;">← Back to Albums</a>
            <div id="album-songs">
                {{SONGS}}
            </div>
        </div>
    </div>
</body>
</html>`

const artistsPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Artists - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 20px; }
        .artist-card { background: #f8f9fa; border-radius: 10px; padding: 20px; text-align: center; text-decoration: none; color: #333; transition: transform 0.2s, box-shadow 0.2s; }
        .artist-card:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
        .artist-name { font-weight: 600; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist" class="active">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Artists</h1>
        <div class="section">
            <div id="artists-grid" hx-get="/admin/library/artist/list" hx-trigger="load">
                {{ARTISTS}}
            </div>
        </div>
    </div>
</body>
</html>`

const artistDetailPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ARTIST_NAME}} - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 18px; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.add-queue { background: #28a745; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .duration { font-family: monospace; }
        .album-links { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 20px; }
        .album-card { background: #f8f9fa; border-radius: 8px; padding: 10px 16px; text-decoration: none; color: #333; font-size: 13px; }
        .album-card:hover { background: #667eea; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist" class="active">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>{{ARTIST_NAME}}</h1>
        <div id="message"></div>
        <div class="section">
            <a href="/admin/library/artist" style="color:#667eea;text-decoration:none;margin-bottom:20px;display:inline-block;">← Back to Artists</a>
            <h2>Discography</h2>
            <div class="album-links">{{ALBUM_LINKS}}</div>
            <h2>All Songs</h2>
            <div id="artist-songs">
                {{SONGS}}
            </div>
        </div>
    </div>
</body>
</html>`

const deletedPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deleted Music - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        button { background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; font-weight: 500; }
        button:hover { background: #5568d3; }
        button.restore { background: #28a745; }
        button.restore:hover { background: #218838; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .actions { display: flex; gap: 6px; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .duration { font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted" class="active">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Deleted Music</h1>
        <div id="message"></div>
        <div class="section">
            <h2>Deleted Songs</h2>
            <p style="color:#666;margin-bottom:15px;">These songs are not in the library. You can re-add them.</p>
            <div id="deleted-table" hx-get="/admin/deleted/songs" hx-trigger="load, refresh">
                {{DELETED}}
            </div>
        </div>
    </div>
</body>
</html>`

const historyPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Play History - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; font-size: 13px; }
        th { background: #f8f9fa; color: #555; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .empty { color: #999; font-style: italic; padding: 20px; text-align: center; }
        .duration { font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history" class="active">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Play History</h1>
        <div class="section">
            <h2>Recently Played</h2>
            <div id="history-table" hx-get="/admin/history/items" hx-trigger="load, every 30s">
                {{HISTORY}}
            </div>
        </div>
    </div>
</body>
</html>`

const editSongPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit Song - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; color: #555; font-weight: 500; font-size: 14px; }
        input { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 5px; font-size: 14px; }
        input:focus { outline: none; border-color: #667eea; }
        button { background: #667eea; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer; font-size: 14px; font-weight: 500; }
        button:hover { background: #5568d3; }
        .error { background: #f8d7da; color: #721c24; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
        .success { background: #d4edda; color: #155724; padding: 10px; border-radius: 5px; margin-bottom: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Edit Song Metadata</h1>
        <div id="message"></div>
        <div class="section">
            <form hx-post="/admin/songs/{{SONG_ID}}/edit" hx-target="#message" hx-swap="innerHTML">
                <div class="form-group">
                    <label for="title">Title</label>
                    <input type="text" id="title" name="title" value="{{TITLE}}" required>
                </div>
                <div class="form-group">
                    <label for="artist">Artist</label>
                    <input type="text" id="artist" name="artist" value="{{ARTIST}}" required>
                </div>
                <div class="form-group">
                    <label for="album">Album</label>
                    <input type="text" id="album" name="album" value="{{ALBUM}}">
                </div>
                <div class="form-group">
                    <label for="genre">Genre</label>
                    <input type="text" id="genre" name="genre" value="{{GENRE}}">
                </div>
                <div class="form-group">
                    <label for="track_number">Track Number</label>
                    <input type="number" id="track_number" name="track_number" value="{{TRACK_NUMBER}}" min="0">
                </div>
                <div class="form-group">
                    <label for="track_total">Track Total</label>
                    <input type="number" id="track_total" name="track_total" value="{{TRACK_TOTAL}}" min="0">
                </div>
                <button type="submit">Save Changes</button>
            </form>
        </div>
    </div>
</body>
</html>`

const analyticsPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Analytics - Radio Admin</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .nav { margin-bottom: 30px; display: flex; gap: 10px; flex-wrap: wrap; }
        .nav a { color: #667eea; text-decoration: none; font-weight: 500; padding: 8px 16px; border-radius: 5px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); font-size: 14px; }
        .nav a:hover, .nav a.active { background: #667eea; color: white; }
        .section { background: white; border-radius: 10px; padding: 30px; margin-bottom: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h2 { color: #333; margin-bottom: 20px; font-size: 22px; }
        .period-selector { display: flex; gap: 10px; margin-bottom: 20px; }
        .period-btn { background: #f8f9fa; color: #333; border: 1px solid #ddd; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 13px; }
        .period-btn.active { background: #667eea; color: white; border-color: #667eea; }
        .chart-container { position: relative; height: 400px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/admin">Dashboard</a>
            <a href="/admin/library">Library</a>
            <a href="/admin/library/album">Albums</a>
            <a href="/admin/library/artist">Artists</a>
            <a href="/admin/playlists">Playlists</a>
            <a href="/admin/deleted">Deleted</a>
            <a href="/admin/history">History</a>
            <a href="/admin/analytics" class="active">Analytics</a>
            <a href="/listen">Listen</a>
        </div>
        <h1>Listener Analytics</h1>
        <div class="section">
            <h2>Listeners Over Time</h2>
            <div class="period-selector">
                <button class="period-btn active" onclick="loadData('hour')">Last Hour</button>
                <button class="period-btn" onclick="loadData('day')">Last 24 Hours</button>
                <button class="period-btn" onclick="loadData('week')">Last Week</button>
                <button class="period-btn" onclick="loadData('month')">Last Month</button>
                <button class="period-btn" onclick="loadData('year')">Last Year</button>
            </div>
            <div class="chart-container">
                <canvas id="listenersChart"></canvas>
            </div>
        </div>
    </div>
    <script>
        var ctx = document.getElementById('listenersChart').getContext('2d');
        var chart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Listeners',
                    data: [],
                    borderColor: '#667eea',
                    backgroundColor: 'rgba(102, 126, 234, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: { beginAtZero: true, ticks: { stepSize: 1 } }
                },
                plugins: {
                    tooltip: {
                        callbacks: {
                            afterLabel: function(context) {
                                return context.raw.songName || '';
                            }
                        }
                    }
                }
            }
        });

        function loadData(period) {
            document.querySelectorAll('.period-btn').forEach(function(btn) {
                btn.classList.remove('active');
            });
            event.target.classList.add('active');

            fetch('/admin/analytics/data?period=' + period)
                .then(function(response) { return response.json(); })
                .then(function(data) {
                    chart.data.labels = data.map(function(d) { return d.time; });
                    chart.data.datasets[0].data = data.map(function(d) { return d.count; });
                    chart.data.datasets[0].label = 'Listeners';
                    chart.update();
                });
        }

        loadData('hour');
    </script>
</body>
</html>`

const songsTableTemplate = `<table>
    <thead>
        <tr><th>Title</th><th>Artist</th><th>Duration</th><th>Actions</th></tr>
    </thead>
    <tbody>%s</tbody>
</table>`

const songRowTemplate = `<tr>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="add-queue" hx-post="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML" hx-on::after-request="htmx.trigger('#queue-table', 'refresh')">Queue</button>
        <button class="delete" hx-delete="/admin/songs/%d" hx-target="#message" hx-swap="innerHTML" hx-confirm="Remove from library?" hx-on::after-request="htmx.trigger('#songs-table', 'refresh')">Remove</button>
    </td>
</tr>`

const emptySongsTemplate = `<div class="empty">No songs in library</div>`

const libraryTableTemplate = `<table>
    <thead>
        <tr><th></th><th>Title</th><th>Artist</th><th>Album</th><th>Genre</th><th>Track</th><th>Duration</th><th>Actions</th></tr>
    </thead>
    <tbody>%s</tbody>
</table>`

const librarySongRowTemplate = `<tr>
    <td><button class="favourite-btn %s" hx-post="/admin/songs/%d/favourite" hx-target="this" hx-swap="outerHTML" hx-on::after-request="htmx.trigger('#library-table', 'refresh')">★</button></td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td>%d</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="add-queue" hx-post="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML">Queue</button>
        <form style="display:inline;" hx-post="/admin/add-to-playlist" hx-target="#message" hx-swap="innerHTML">
            <input type="hidden" name="song_id" value="%d">
            <select name="playlist_id" onchange="this.form.dispatchEvent(new Event('submit'))" style="padding:4px;font-size:12px;">
                <option value="">+ Playlist</option>
                %s
            </select>
        </form>
        <a href="/admin/songs/%d/edit" style="padding:4px 8px;font-size:12px;color:#667eea;">Edit</a>
        <button class="delete" hx-post="/admin/library/%d/remove" hx-target="#message" hx-swap="innerHTML" hx-confirm="Remove from library?" hx-on::after-request="htmx.trigger('#library-table', 'refresh')">Remove</button>
    </td>
</tr>`

const emptyLibraryTemplate = `<div class="empty">No songs in library</div>`

const playlistsTableTemplate = `<table>
    <thead><tr><th>Name</th><th>Songs</th><th>Actions</th></tr></thead>
    <tbody>%s</tbody>
</table>`

const playlistRowTemplate = `<tr>
    <td><a href="/admin/playlists/%d" style="color:#667eea;text-decoration:none;font-weight:500;">%s</a></td>
    <td>%d songs</td>
    <td class="actions">
        <button class="delete" hx-delete="/admin/playlists/%d" hx-target="#message" hx-swap="innerHTML" hx-confirm="Delete playlist?" hx-on::after-request="htmx.trigger('#playlists-table', 'refresh')">Delete</button>
    </td>
</tr>`

const emptyPlaylistsTemplate = `<div class="empty">No playlists yet. Create one above!</div>`

const playlistSongsTableTemplate = `<table>
    <thead>
        <tr><th></th><th>#</th><th>Title</th><th>Artist</th><th>Album</th><th>Genre</th><th>Duration</th><th>Actions</th></tr>
    </thead>
    <tbody id="playlist-songs-tbody">%s</tbody>
</table>`

const playlistSongRowTemplate = `<tr data-song-id="%d" data-playlist-id="%d">
    <td><span class="drag-handle">☰</span></td>
    <td>%d</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="add-queue" hx-post="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML">Queue</button>
        <button class="delete" hx-post="/admin/playlists/%d/remove/%d" hx-target="#message" hx-swap="innerHTML" hx-confirm="Remove from playlist?" hx-on::after-request="htmx.trigger('#playlist-songs', 'refresh')">Remove</button>
    </td>
</tr>`

const deletedTableTemplate = `<table>
    <thead><tr><th>Title</th><th>Artist</th><th>Album</th><th>Genre</th><th>Duration</th><th>Actions</th></tr></thead>
    <tbody>%s</tbody>
</table>`

const deletedSongRowTemplate = `<tr>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="restore" hx-post="/admin/library/%d/add" hx-target="#message" hx-swap="innerHTML" hx-on::after-request="htmx.trigger('#deleted-table', 'refresh')">Add to Library</button>
    </td>
</tr>`

const emptyDeletedTemplate = `<div class="empty">No deleted songs</div>`

const historyTableTemplate = `<table>
    <thead><tr><th>Played At</th><th>Title</th><th>Artist</th><th>Album</th><th>Genre</th><th>Duration</th><th>Played</th></tr></thead>
    <tbody>%s</tbody>
</table>`

const historyRowTemplate = `<tr>
    <td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="duration">%s</td><td class="duration">%s</td>
</tr>`

const emptyHistoryTemplate = `<div class="empty">No songs played yet</div>`

const queueTableTemplate = `<table>
    <thead>
        <tr><th></th><th>Pos</th><th>Track</th><th>Title</th><th>Artist</th><th>Album</th><th>Duration</th><th>Actions</th></tr>
    </thead>
    <tbody id="queue-tbody">%s</tbody>
</table>`

const queueRowTemplate = `<tr data-queue-id="%d" data-position="%d">
    <td><span class="drag-handle">☰</span></td>
    <td>%d</td>
    <td>%d</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%ds</td>
    <td class="actions">
        <button class="delete" hx-delete="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML" hx-on::after-request="htmx.trigger('#queue-table', 'refresh')">Remove</button>
    </td>
</tr>`

const emptyQueueTemplate = `<div class="empty">Queue is empty</div>`

const nowPlayingTemplate = `<div class="now-playing">
    %s
    <h3>Now Playing</h3>
    <p><strong>%s</strong> by %s</p>
    <p>%s</p>
    <p>Duration: <span class="duration">%s</span></p>
    <p>Status: <span class="status %s">%s</span></p>
    <form hx-post="/admin/play" hx-target="#now-playing" hx-swap="innerHTML">
        <button type="submit" class="%s">%s</button>
    </form>
</div>`

const nowPlayingEmptyTemplate = `<div class="now-playing">
    <h3>Now Playing</h3>
    <p>No song currently playing</p>
    <p>Status: <span class="status %s">%s</span></p>
    <form hx-post="/admin/play" hx-target="#now-playing" hx-swap="innerHTML">
        <button type="submit" class="%s">%s</button>
    </form>
</div>`

const albumSongsTableTemplate = `<table>
    <thead><tr><th>#</th><th>Title</th><th>Artist</th><th>Genre</th><th>Duration</th><th>Actions</th></tr></thead>
    <tbody>%s</tbody>
</table>`

const albumSongRowTemplate = `<tr>
    <td>%d</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="add-queue" hx-post="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML">Queue</button>
    </td>
</tr>`

const emptyAlbumsTemplate = `<div class="empty">No albums found</div>`

const artistSongsTableTemplate = `<table>
    <thead><tr><th>Album</th><th>#</th><th>Title</th><th>Genre</th><th>Duration</th><th>Actions</th></tr></thead>
    <tbody>%s</tbody>
</table>`

const artistSongRowTemplate = `<tr>
    <td>%s</td>
    <td>%d</td>
    <td>%s</td>
    <td>%s</td>
    <td class="duration">%s</td>
    <td class="actions">
        <button class="add-queue" hx-post="/admin/queue/%d" hx-target="#message" hx-swap="innerHTML">Queue</button>
    </td>
</tr>`

const emptyArtistsTemplate = `<div class="empty">No artists found</div>`

const messageSuccessTemplate = `<div class="success">%s</div>`
const messageErrorTemplate = `<div class="error">%s</div>`
