// Exported pages report completion; the parent owns routes and choices.
(function () {
  function send(type, fields) {
    parent.postMessage(Object.assign({channel: 'galgame-content', type: type}, fields || {}), '*');
  }
  function init() {
    var videos = Array.from(document.querySelectorAll('video'));
    var finished = false;
    function complete() {
      if (!finished) { finished = true; send('ended'); }
    }
    videos.forEach(function (video) {
      video.addEventListener('ended', function () {
        if (videos.every(function (v) { return v.ended; })) { complete(); }
      });
      video.addEventListener('error', function () { send('error'); });
      if (video.error) { send('error'); }
    });
    // Authored interactive pages can explicitly finish without a video.
    window.addEventListener('galgame:complete', complete);
    var playbackStarted = false;
    window.addEventListener('message', function (e) {
      if (e.source !== parent || !e.data || e.data.channel !== 'galgame-player' || e.data.type !== 'volume') { return; }
      videos.forEach(function (video) {
        if (typeof e.data.volume === 'number' && Number.isFinite(e.data.volume)) {
          video.volume = Math.max(0, Math.min(1, e.data.volume));
        }
        video.muted = !!e.data.muted;
        if (!playbackStarted) {
          // Browser autoplay restrictions still permit manual playback.
          video.play().catch(function () { video.controls = true; });
        }
      });
      playbackStarted = true;
    });
    send('ready', {hasVideo: videos.length > 0});
  }
  if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', init, {once: true}); }
  else { init(); }
})();
