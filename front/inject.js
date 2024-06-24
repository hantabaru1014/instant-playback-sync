export default (host, roomId) => {
  (function () {
    'use strict';

    let targetMediaElement = null;
    let ws = null;
    let lastReceivedSyncMsgTime = null;

    const log = (msg) => console.log(`[instant-playback-sync] ${msg}`);
    const error = (msg) => console.error(`[instant-playback-sync] ${msg}`);

    const connectWS = (wsAddress) => {
      ws = new WebSocket(wsAddress);

      ws.onmessage = (msg) => {
        log(`Received message: ${msg.data}`);
        const data = JSON.parse(msg.data);
        switch (data.cmd) {
          case 'sync':
            handleSyncEvent(data);
            lastReceivedSyncMsgTime = Date.now();
            break;
          case 'reqSync':
            sendSyncEvent();
            break;
          default:
            error('Unknown command:', data.cmd);
        }
      }
    }

    const sendSyncEvent = () => {
      if (Date.now() - lastReceivedSyncMsgTime < 500) {
        // たぶん送られてきたsync cmdによる再生制御なので、送信しない
        return;
      }
      const eventType = targetMediaElement.paused ? 'pause' : 'play';
      const msg = {
        cmd: 'sync',
        p: {
          pageUrl: window.location.href,
          event: eventType,
          currentTime: targetMediaElement.currentTime,
          playbackRate: targetMediaElement.playbackRate,
        }
      }
      ws.send(JSON.stringify(msg));
    }

    const handleSyncEvent = (msg) => {
      const { event, currentTime, playbackRate } = msg.p;
      // 再生時間の差が1秒未満ならぷつぷつしないように値を入れない
      if (currentTime > 0 && Math.abs(targetMediaElement.currentTime - currentTime) > 1) {
        targetMediaElement.currentTime = currentTime;
      }
      targetMediaElement.playbackRate = playbackRate;
      switch (event) {
        case 'play':
          if (targetMediaElement.paused) {
            targetMediaElement.play();
          }
          break;
        case 'pause':
          if (!targetMediaElement.paused) {
            targetMediaElement.pause();
          }
          break;
        default:
          log('Unknown event type:', event);
      }
    }

    const getMediaElement = () => {
      const elms = document.getElementsByTagName('video');
      if (elms.length === 0) {
        error('No video element found');
        return null;
      }else{
        return elms[0];
      }
    }

    const setVideoEvents = (mediaElm) => {
      mediaElm.addEventListener('play', () => {
        log('Video started playing');
        sendSyncEvent();
      });
      mediaElm.addEventListener('pause', () => {
        log('Video paused');
        sendSyncEvent();
      });
      mediaElm.addEventListener('seeked', () => {
        // 再生時間の変更
        log('Video seeked');
        sendSyncEvent();
      });
      mediaElm.addEventListener('ratechange', () => {
        log('Video rate changed');
        sendSyncEvent();
      });
    }

    const main = () => {
      targetMediaElement = getMediaElement();
      if (!targetMediaElement) {
        return;
      }
      setVideoEvents(targetMediaElement);
      connectWS(`wss://${host}/api/rooms/${roomId}/ws`);
    }

    log(`script loaded for room: ${roomId}`);
    main();
  })();
};