#!/bin/bash
systemctl daemon-reload
systemctl enable cacophony-config-import
/usr/bin/cacophony-config-import
