#include <ctype.h>
#include <stdlib.h>
#include <stdio.h>

#include <mach/mach_port.h>
#include <mach/mach_interface.h>
#include <mach/mach_init.h>

#include <IOKit/pwr_mgt/IOPMLib.h>
#include <IOKit/IOMessage.h>

io_connect_t root_port; // a reference to the Root Power Domain IOService
// notification port allocated by IORegisterForSystemPower
IONotificationPortRef notifyPortRef;
// notifier object, used to deregister later
io_object_t notifierObject;
// this parameter is passed to the callback
void *refCon;
CFRunLoopRef runLoop;

void SleepCallBack(void *refCon, io_service_t service, natural_t messageType, void *messageArgument)
{
    switch (messageType)
    {
    case kIOMessageSystemWillSleep:
        NotifySleep();
        IOAllowPowerChange(root_port, (long)messageArgument);
        break;
    case kIOMessageSystemWillPowerOn:
        NotifyWake();
        break;
    case kIOMessageSystemHasPoweredOn:
        break;
    default:
        break;
    }
}

void registerNotifications()
{
    root_port = IORegisterForSystemPower(refCon, &notifyPortRef, SleepCallBack, &notifierObject);
    if (root_port == 0)
    {
        printf("IORegisterForSystemPower failed\n");
    }

    CFRunLoopAddSource(CFRunLoopGetCurrent(),
                       IONotificationPortGetRunLoopSource(notifyPortRef), kCFRunLoopCommonModes);

    runLoop = CFRunLoopGetCurrent();
    CFRunLoopRun();
}

void unregisterNotifications()
{
    CFRunLoopRemoveSource(runLoop,
                          IONotificationPortGetRunLoopSource(notifyPortRef),
                          kCFRunLoopCommonModes);

    IODeregisterForSystemPower(&notifierObject);

    IOServiceClose(root_port);

    IONotificationPortDestroy(notifyPortRef);

    CFRunLoopStop(runLoop);
}
