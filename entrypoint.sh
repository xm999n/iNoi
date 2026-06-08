#!/bin/sh

umask ${UMASK}

if [ "$1" = "version" ]; then
  ./iNoi version
else
  # Check file of /opt/inoi/data permissions for current user
  # Check whether the current user can write and execute the data directory.
  if [ -d ./data ]; then
    if ! [ -w ./data ] || ! [ -x ./data ]; then
  cat <<EOF
Error: Current user does not have write and/or execute permissions for the ./data directory: $(pwd)/data
Please check the permissions of the mounted iNoi data directory for more information.
错误：当前用户没有 ./data 目录（$(pwd)/data）的写和/或执行权限。
请检查挂载的 iNoi 数据目录权限。
Exiting...
EOF
      exit 1
    fi
  fi

  chown -R ${PUID}:${PGID} /opt/inoi/
  exec su-exec ${PUID}:${PGID} ./iNoi server --no-prefix
fi
