(() => {
  'use strict';

  const menuButton = document.querySelector('.menu-toggle');
  const navigation = document.querySelector('.navigation');
  const toast = document.querySelector('.toast');
  const tabs = [...document.querySelectorAll('[data-step]')];
  let toastTimeout;

  // All content and localized labels are rendered by Hugo. This file only
  // enhances native HTML controls, never selects or replaces a language.
  function selectTab(selected) {
    tabs.forEach(tab => {
      const active = tab === selected;
      tab.classList.toggle('active', active);
      tab.setAttribute('aria-selected', String(active));
      tab.tabIndex = active ? 0 : -1;
      document.getElementById(tab.getAttribute('aria-controls')).hidden = !active;
    });
  }

  function setMenu(open, returnFocus = false) {
    if (!menuButton || !navigation) return;
    navigation.classList.toggle('open', open);
    menuButton.setAttribute('aria-expanded', String(open));
    menuButton.setAttribute('aria-label', open ? menuButton.dataset.closeLabel : menuButton.dataset.openLabel);
    if (returnFocus) menuButton.focus();
  }

  async function copyText(text) {
    try {
      if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      const previousFocus = document.activeElement;
      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.setAttribute('readonly', '');
      textarea.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0';
      document.body.append(textarea);
      textarea.select();
      let copied = false;
      try { copied = document.execCommand('copy'); } catch { /* Show the rendered fallback message. */ }
      textarea.remove();
      previousFocus?.focus({ preventScroll: true });
      return copied;
    }
  }

  document.addEventListener('click', async event => {
    const button = event.target.closest('[data-copy-target]');
    if (!button) return;
    const text = document.getElementById(button.dataset.copyTarget)?.textContent;
    if (!text) return;
    button.disabled = true;
    const success = await copyText(text.trim());
    button.disabled = false;
    if (!toast) return;
    clearTimeout(toastTimeout);
    toast.textContent = success ? toast.dataset.copySuccess : toast.dataset.copyFailure;
    toast.classList.add('visible');
    toastTimeout = setTimeout(() => toast.classList.remove('visible'), 2800);
  });

  tabs.forEach((tab, index) => {
    tab.addEventListener('click', () => selectTab(tab));
    tab.addEventListener('keydown', event => {
      let next;
      if (event.key === 'ArrowDown' || event.key === 'ArrowRight') next = (index + 1) % tabs.length;
      if (event.key === 'ArrowUp' || event.key === 'ArrowLeft') next = (index + tabs.length - 1) % tabs.length;
      if (event.key === 'Home') next = 0;
      if (event.key === 'End') next = tabs.length - 1;
      if (next === undefined) return;
      event.preventDefault();
      tabs[next].focus();
      selectTab(tabs[next]);
    });
  });
  if (tabs.length) selectTab(tabs[0]);

  menuButton?.addEventListener('click', () => setMenu(menuButton.getAttribute('aria-expanded') !== 'true'));
  navigation?.querySelectorAll('a').forEach(link => link.addEventListener('click', () => setMenu(false)));
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && navigation?.classList.contains('open')) setMenu(false, true);
  });
  document.addEventListener('click', event => {
    if (!event.target.closest('.site-header')) setMenu(false);
  });
  window.matchMedia('(max-width: 940px)').addEventListener('change', () => setMenu(false));

  const docLinks = [...document.querySelectorAll('.docs-sidebar nav a')];
  if (docLinks.length && 'IntersectionObserver' in window) {
    const observer = new IntersectionObserver(entries => {
      const entry = entries.find(item => item.isIntersecting);
      if (!entry) return;
      docLinks.forEach(link => {
        const current = link.hash === `#${entry.target.id}`;
        link.classList.toggle('current', current);
        if (current) link.setAttribute('aria-current', 'location');
        else link.removeAttribute('aria-current');
      });
    }, { rootMargin: '-15% 0px -60% 0px', threshold: 0 });
    document.querySelectorAll('.docs-prose h2[id]').forEach(heading => observer.observe(heading));
  }

  document.documentElement.classList.remove('no-js');
})();
