package wifi

/*
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <linux/wireless.h>
#include <arpa/inet.h>
#include <linux/if_ether.h>

const char ErrInvalidArg[] = "Invalid argument";

// return NULL on success, on error the result of 'Man strerror'
// TODO: pass iw struct + connected socket as argument
const char *
set_channel(const char *iface, int channel)
{
  struct iwreq iw;
  size_t iface_len;
  int fd, r;

  if (!iface || (iface_len = strlen(iface)) > IFNAMSIZ)
    return ErrInvalidArg;

  strncpy(iw.ifr_ifrn.ifrn_name, iface, iface_len);

  fd = socket(PF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
  if (fd < 0)
    return strerror(errno);

  iw.u.freq = (struct iw_freq){
      // Note: the 'freq' field is overloaded with frequency + channel
      // -> low numbers N<=1000 are encoded as e(xponent)=0 and m(antissa)=N
      .m = channel,
      .e = 0};
  r = ioctl(fd, SIOCSIWFREQ, &iw);
  close(fd);
  if (r == -1)
    return strerror(errno);
  return NULL;
}
*/
import "C"

import (
	"context"
	"errors"
	"log"
	"time"
	"unsafe"
)

// set the channel (frequency) for the interface <ifi> to <ch>
func SetChannel(ifi string, ch int) error {
	cifi := C.CString(ifi)
	defer C.free(unsafe.Pointer(cifi))
	cp := C.set_channel(cifi, C.int(ch))
	if cp != nil {
		return errors.New(C.GoString(cp))
	}
	return nil
}

// go through channels 1 to 13 until <-done
func HopChannels(ctx context.Context, ifname string) {
	var (
		channel int
		tick    *time.Ticker
		err     error
	)
	tick = time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			log.Printf("channel = %d", channel+1)
			err = SetChannel(ifname, channel+1)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			channel = (channel + 1) % 13
		case <-ctx.Done():
			return
		}
	}
}
