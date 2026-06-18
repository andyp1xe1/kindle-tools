INSERT OR IGNORE INTO interfaces(interface) VALUES('application');
INSERT OR IGNORE INTO handlerIds(handlerId) VALUES('{{APP_ID}}');
INSERT OR REPLACE INTO properties(handlerId,name,value) VALUES('{{APP_ID}}','lipcId','{{APP_ID}}');
INSERT OR REPLACE INTO properties(handlerId,name,value) VALUES('{{APP_ID}}','command','/usr/bin/mesquite -l {{APP_ID}} -c file://{{WIDGET_DIR}}/');
INSERT OR REPLACE INTO properties(handlerId,name,value) VALUES('{{APP_ID}}','supportedOrientation','U');
