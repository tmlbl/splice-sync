import os
import sqlite3
from pathlib import Path

path = Path('/Users/' + os.getlogin() + '/AppData/Local/SpliceSettings/users/default/')
splice_users = os.listdir(path)
sounds_db_path = path.joinpath(splice_users[0], 'sounds.db')

db = sqlite3.connect(sounds_db_path)

structure = {
    "drums": [
		"808",
		"kicks",
		"snares",
		"rims",
		"claps",
		"hats",
		"toms",
		"snaps",
		"percussion"
    ],
    "vocals": [
        "female",
        "male"
    ],
    "plucks": None,
    "bells": None,
    "fx": [
        "foley"
    ],
    "live sounds": [
        "percussion"
    ],
    "percussion": None,
    "synth": None,
    "ambient": None
}

target_dir = Path('C:\\Users\\timle\\Music\\_SpliceSorted')

cur = db.cursor().execute('select sample_type, local_path, tags from samples;')
could_not_sort = 0
for c in cur:
    sample_type = c[0]
    local_path = c[1]
    if local_path is None:
        print(c)
        continue
    file_name = os.path.basename(local_path)
    tags = set(c[2].split(','))

    print(file_name)
    if sample_type == "loop":
        continue

    sorted_dir = None
    for k in structure:
        if k in tags:
            sorted_dir = k
            v = structure[k]
            if type(v) is list:
                for vv in v:
                    if vv in tags:
                        sorted_dir = k + '/' + vv
            break
    
    if sorted_dir is None:
        # print('Could not sort tags %s' % tags)
        could_not_sort += 1
        continue
    
    sorted_path = target_dir.joinpath(sorted_dir, file_name)
    if os.path.isfile(sorted_path):
        continue
    os.makedirs(target_dir.joinpath(sorted_dir), exist_ok=True)
    os.link(local_path, sorted_path)
    # print(local_path, sorted_path)

print('Could not sort %d samples' % could_not_sort)
