(function () {
  'use strict';

  const WORD_LIST = [
    ['CRANE', 'CANER', 'NACRE', 'OCEAN'],
    ['STALE', 'TALES', 'TESLA', 'LEAST', 'SLATE'],
    ['NOTES', 'STONE', 'TONES', 'ONSET'],
    ['PARES', 'REAPS', 'SPARE', 'SPEAR'],
    ['LEMON', 'MELON', 'ENROL'],
    ['LISTEN', 'SILENT', 'TINSEL', 'ENLIST', 'INLETS'],
    ['MASTER', 'STREAM', 'REMAST', 'TAMERS'],
    ['GARDEN', 'RANGED', 'DANGER', 'GANDER'],
    ['SECURE', 'CEREUS', 'RESCUE'],
    ['LUSTER', 'RESULT', 'RUSTLE', 'SUTLER'],
  ];

  const ROUNDS = 5;
  let round = 0;
  let score = 0;
  let currentSet = [];
  let scrambled = '';
  let inputValue = '';
  let found = new Set();
  let message = '';
  let messageOk = true;
  let gameOver = false;

  function shuffle(s) {
    return s.split('').sort(function() { return Math.random() - 0.5; }).join('');
  }

  function startRound() {
    const setIdx = Math.floor(Math.random() * WORD_LIST.length);
    currentSet = WORD_LIST[setIdx].slice();
    const longest = currentSet.reduce(function(a, b) { return a.length >= b.length ? a : b; });
    scrambled = shuffle(longest);
    found = new Set();
    inputValue = '';
    message = '';
    render();
  }

  function submit() {
    const word = inputValue.toUpperCase().trim();
    inputValue = '';
    if (!word || gameOver) { return; }

    if (found.has(word)) {
      message = 'Already found "' + word + '"!';
      messageOk = false;
    } else if (currentSet.includes(word)) {
      found.add(word);
      score += word.length * 10;
      message = '✓ "' + word + '" (+' + (word.length * 10) + ')';
      messageOk = true;
      if (found.size === currentSet.length) {
        score += 50;
        message = '🎉 All words found! +50 bonus';
        setTimeout(nextRound, 1200);
      }
    } else {
      message = '"' + word + '" is not in the list';
      messageOk = false;
    }
    render();
  }

  function nextRound() {
    if (gameOver) return;
    round++;
    if (round >= ROUNDS) {
      endGame();
    } else {
      startRound();
    }
  }

  function endGame() {
    gameOver = true;
    render();
    setTimeout(function() { window.GameShelf.gameOver(score); }, 400);
  }

  function render() {
    const root = document.getElementById('game-root');
    root.innerHTML = '';
    root.className = 'flex flex-col items-center gap-4 w-full max-w-md mx-auto py-4';

    // Header
    const header = document.createElement('div');
    header.className = 'flex justify-between w-full text-sm text-gray-500';
    header.innerHTML = '<span>Round ' + (round + 1) + ' / ' + ROUNDS + '</span>' +
      '<span>Score: <strong class="text-gray-900">' + score + '</strong></span>';
    root.appendChild(header);

    // Scrambled letters
    const scrambleEl = document.createElement('div');
    scrambleEl.className = 'flex gap-2 flex-wrap justify-center my-2';
    scrambled.split('').forEach(function(ch) {
      const s = document.createElement('span');
      s.textContent = ch;
      s.className = 'w-10 h-10 flex items-center justify-center bg-blue-100 text-blue-900 font-bold text-xl rounded-lg border border-blue-200';
      scrambleEl.appendChild(s);
    });
    root.appendChild(scrambleEl);

    // Hint
    const hint = document.createElement('p');
    hint.className = 'text-gray-400 text-xs';
    hint.textContent = 'Find all ' + currentSet.length + ' words using these letters';
    root.appendChild(hint);

    // Input row
    const inputRow = document.createElement('div');
    inputRow.className = 'flex gap-2 w-full';
    const inp = document.createElement('input');
    inp.type = 'text';
    inp.value = inputValue;
    inp.maxLength = 10;
    inp.placeholder = 'Type a word...';
    inp.className = 'flex-1 border border-gray-200 rounded-lg px-4 py-2 text-sm font-mono uppercase focus:outline-none focus:ring-2 focus:ring-blue-300';
    inp.addEventListener('input', function(e) { inputValue = e.target.value; });
    inp.addEventListener('keydown', function(e) { if (e.key === 'Enter') submit(); });
    inputRow.appendChild(inp);

    const btn = document.createElement('button');
    btn.textContent = 'Submit';
    btn.className = 'px-5 py-2 rounded-lg bg-blue-600 text-white font-semibold text-sm hover:bg-blue-700';
    btn.addEventListener('click', submit);
    inputRow.appendChild(btn);
    root.appendChild(inputRow);
    setTimeout(function() { inp.focus(); }, 0);

    // Message
    if (message) {
      const msg = document.createElement('p');
      msg.textContent = message;
      msg.className = 'text-sm font-medium ' + (messageOk ? 'text-green-600' : 'text-red-500');
      root.appendChild(msg);
    }

    // Found words display
    const foundEl = document.createElement('div');
    foundEl.className = 'flex flex-wrap gap-2 justify-center';
    currentSet.forEach(function(w) {
      const span = document.createElement('span');
      span.textContent = found.has(w) ? w : '?'.repeat(w.length);
      span.className = 'px-3 py-1 rounded-full text-xs font-mono border ' +
        (found.has(w)
          ? 'bg-green-100 text-green-700 border-green-300'
          : 'bg-gray-100 text-gray-400 border-gray-200');
      foundEl.appendChild(span);
    });
    root.appendChild(foundEl);

    if (!gameOver) {
      const skipBtn = document.createElement('button');
      skipBtn.textContent = round < ROUNDS - 1 ? 'Skip round →' : 'Finish';
      skipBtn.className = 'text-xs text-gray-400 hover:text-gray-600 mt-2';
      skipBtn.addEventListener('click', nextRound);
      root.appendChild(skipBtn);
    }
  }

  startRound();
})();
