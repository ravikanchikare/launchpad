#ifndef HARNEZPAD_MENUBAR_DARWIN_H
#define HARNEZPAD_MENUBAR_DARWIN_H

void harnezpadInstallMenuBar(void *window);
void harnezpadUninstallMenuBar(void);
void harnezpadShowSettings(void);
void harnezpadCheckForUpdates(void);
void harnezpadShowAbout(void);
void harnezpadShowHelp(void);
void harnezpadPresentAbout(void);
void harnezpadPresentUpdateAlert(const char *title, const char *message);
int harnezpadPresentUpdateConfirm(const char *title, const char *message);
void harnezpadQuit(void);
void harnezpadToggleSidebar(void);

#endif
