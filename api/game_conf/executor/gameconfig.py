# -*- coding: utf-8 -*-
"""给 tabtoy 生成的 gameconfig.proto 注入 go_package。

tabtoy 每次重生成 proto 都会抹掉 go_package，必须在导出流程中紧随其后运行。
路径相对于本文件解析，可在任意 cwd 下执行。
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))
PROTO_FILE = os.path.normpath(os.path.join(HERE, '..', '..', 'protocol', 'src', 'gameconfig.proto'))
GO_PACKAGE = 'server.slg.com/api/protocol/pb/pb_gameconfig'

with open(PROTO_FILE, 'r', encoding='utf-8') as f:
    lines = f.readlines()

if not any('option go_package' in line for line in lines):
    # 插到 package 声明行之后（tabtoy 生成物必有 package）
    insert_at = 0
    for i, line in enumerate(lines):
        if line.startswith('package '):
            insert_at = i + 1
            break
    lines.insert(insert_at, 'option go_package = "%s";\n' % GO_PACKAGE)
    with open(PROTO_FILE, 'w', encoding='utf-8') as f:
        f.writelines(lines)
    print('Added go_package to %s' % PROTO_FILE)
else:
    print('go_package already present in %s' % PROTO_FILE)
