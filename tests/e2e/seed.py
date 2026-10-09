"""Only seeds a marked temporary fixture directory created by playback.sh."""
import json, os, pathlib, sqlite3, sys, time
root = pathlib.Path(os.environ['PLAYBACK_TEST_ROOT']).resolve()
if root.parent != pathlib.Path('/tmp') or not root.name.startswith('alreadyon-playback-e2e.') or not (root / '.synthetic-test').is_file():
    raise SystemExit('Refusing to seed anything except the disposable playback fixture')
c = sqlite3.connect(root / 'data/alreadyon.db')
now = int(time.time())
c.execute("UPDATE folder_settings SET tv_paths=?,movie_paths='[]',commercial_paths='[]' WHERE id=1", (json.dumps([str(root / 'media')]),))
for channel in (998, 999):
    c.execute('INSERT OR IGNORE INTO channels(id,number,name) VALUES(?,?,?)', (channel,channel,'Synthetic channel '+str(channel)))
for media, name, duration, kind in ((999,'program.mp4',120,'media'), (997,'ad.avi',12,'commercial'), (996,'ad-red.avi',12,'commercial'), (995,'ad-green.avi',12,'commercial'), (994,'hd.mp4',60,'media'), (993,'multi.mkv',120,'media')):
    path = root / 'media' / name
    c.execute('INSERT OR IGNORE INTO media_files(id,root_kind,path,size_bytes,modified_unix,first_seen_unix,last_seen_unix,duration_seconds,status) VALUES(?,?,?,?,?,0,0,?,?)', (media,kind,str(path),path.stat().st_size,int(path.stat().st_mtime),duration,'ok'))
c.execute("INSERT OR IGNORE INTO identified_items(id,media_file_id,media_type,title,is_manual_correction) VALUES(999,999,'movie','Synthetic Program',1)")
c.execute("UPDATE channels SET hidden=CASE WHEN id IN (998,999) THEN 0 ELSE 1 END, visibility=CASE WHEN id IN (998,999) THEN 'show' ELSE 'hide' END")
c.execute('DELETE FROM schedule_entries WHERE channel_id IN (998,999)')
mode = sys.argv[1] if len(sys.argv) > 1 else ''
# The ladder is opt-in; only its own scenario turns it on.
c.execute('UPDATE playback_settings SET ladder=? WHERE id=1', (1 if mode == 'ladder' else 0,))
entries = [(999,'program',0,now-20,now+90)]
if mode in ('audio', 'audio-commercial'):
    entries = [(993,'commercial' if mode == 'audio-commercial' else 'program',0,now-20,now+90)]
    if mode == 'audio-commercial':
        # Guide cells omit commercials; retain a following program so the channel can be tuned.
        entries.append((999,'program',0,now+90,now+180))
elif mode == 'transitions':
    entries = [(999,'program',0,now-20,now+18),(997,'commercial',0,now+18,now+30),(999,'program',38,now+30,now+70)]
elif mode == 'split':
    # A program split in two with no break, so both halves can direct play.
    entries = [(999,'program',0,now-20,now+14),(999,'program',34,now+14,now+60)]
elif mode in ('quality', 'ladder'):
    # A taller source, so a quality ceiling below it forces a conversion.
    entries = [(994,'program',0,now-20,now+55)]
elif mode == 'guide':
    # Long entries, so their boxes are wide enough to check against the header columns.
    entries = [(999,'program',0,now-1200,now+2400),(999,'program',0,now+2400,now+3300),(999,'program',0,now+3300,now+7200)]
elif mode == 'short':
    # Consecutive ads shorter than the four-second presentation delay.
    entries = [(999,'program',0,now-20,now+16),(996,'commercial',0,now+16,now+18),(995,'commercial',0,now+18,now+20),(999,'program',36,now+20,now+60)]
for channel in (998,999):
    for media,kind,offset,start,end in entries:
        c.execute('INSERT INTO schedule_entries(channel_id,media_file_id,kind,source_offset_seconds,duration_seconds,starts_at_unix,ends_at_unix) VALUES(?,?,?,?,?,?,?)',(channel,media,kind,offset,end-start,start,end))
c.commit()
