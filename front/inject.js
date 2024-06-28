export default (host, roomId) => {
  (function () {
    'use strict';

    let targetMediaElement = null;
    let statusLabel = null;

    let lastReceivedSyncMsgTime = null;

    let postMessage = null;

    const log = (msg) => console.log(`[instant-playback-sync] ${msg}`);
    const error = (msg) => console.error(`[instant-playback-sync] ${msg}`);

    const sendSyncEvent = () => {
      if (Date.now() - lastReceivedSyncMsgTime < 500) {
        // たぶん送られてきたsync cmdによる再生制御なので、送信しない
        return;
      }
      const eventType = targetMediaElement.paused ? 'pause' : 'play';
      postMessage({
        cmd: 'sync',
        p: {
          pageUrl: window.location.href,
          event: eventType,
          currentTime: targetMediaElement.currentTime,
          playbackRate: targetMediaElement.playbackRate,
        }
      });
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

    const hookVideoEvents = (mediaElm) => {
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

    const createWSProxy = () => {
      return new Promise((resolve, reject) => {
        updateStatusLabel('接続中...', 'orange');
        window.addEventListener('message', (event) => {
          if (event.origin !== `https://${host}`) {
            return;
          }
          switch (event.data.cmd) {
            case 'iframe:onLoadedProxy':
              postMessage({
                cmd: 'iframe:connect',
                p: `wss://${host}/api/rooms/${roomId}/ws`
              });
              break;
            case 'iframe:onConnected':
              log('Connected to ws server');
              updateStatusLabel('OK', 'green');
              resolve();
              break;
            case 'iframe:onDisconnected':
            case 'iframe:onError':
              error('Disconnected from ws server');
              updateStatusLabel('同期エラー. 再接続してください!', 'red');
              break;
            default:
              handleWSMessage(event.data);
              break;
          }
        });
  
        const iframe = document.createElement('iframe');
        iframe.src = `https://${host}/wsproxy`;
        iframe.style.display = 'none';
        iframe.id = 'instant-playback-sync-wsproxy';
  
        postMessage = (data) => iframe.contentWindow.postMessage(data, `https://${host}`);
  
        document.body.appendChild(iframe);
      });
    }

    const existsWSProxy = () => {
      return !!document.getElementById('instant-playback-sync-wsproxy');
    }

    const handleWSMessage = (data) => {
      log(`Received message: ${JSON.stringify(data)}`)
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

    const createStatusLabel = (mediaElm) => {
      const label = document.createElement('div');
      label.id = 'instant-playback-sync-status-label';
      label.style.position = 'absolute';
      label.style.zIndex = 9999;
      label.style.left = '10px';
      label.style.top = '10px';

      mediaElm.parentElement.appendChild(label);
      statusLabel = label;
    }

    const updateStatusLabel = (msg, color) => {
      if (!statusLabel) return;
      statusLabel.innerText = msg;
      statusLabel.style.color = color;
    }

    const main = async () => {
      if (existsWSProxy()){
        alert('[instant-playback-sync] このページはすでに同期されています。正常に動作していない場合はページをリロードしてから再度お試しください。');
        return;
      }
      targetMediaElement = getMediaElement();
      if (!targetMediaElement) {
        window.location.href = `https://${host}/r/${roomId}`;
        return;
      }
      createStatusLabel(targetMediaElement);
      
      await createWSProxy();
      hookVideoEvents(targetMediaElement);
    }

    log(`script loaded for room: ${roomId}`);
    main();
  })();
};