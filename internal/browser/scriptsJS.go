package browser

// JS script to highlight clicked elements
const HighlightClickScript = `(e) => e.style.border = "3px solid #00FF00"`

// JS script to highlight typed elements
const HighlightTypeScript = `(e) => e.style.border = "3px solid blue"`

const ScrollDownScript = `() => { window.scrollBy(0, window.innerHeight * 0.7); return true; }`

const ScrollUpScript = `() => { window.scrollBy(0, -window.innerHeight * 0.7); return true; }`

const ObserveElementsScript = `function() {
    const MAX_ITEMS = 600;

    // --- 1. ОЧИСТКА ---
    document.querySelectorAll('[data-agent-id]').forEach(el => el.removeAttribute('data-agent-id'));
    const oldContainer = document.getElementById('agent-ids-overlay');
    if (oldContainer) oldContainer.remove();

    const items = [];
    let idCounter = 1;
    const seen = new Set();

    function isVisible(el) {
        const rect = el.getBoundingClientRect();
        if (rect.width < 1 || rect.height < 1) return false;
        const style = window.getComputedStyle(el);
        return style.visibility !== 'hidden' && style.display !== 'none' && style.opacity !== '0';
    }

    const all = document.body.querySelectorAll('*');
    
    for (const el of all) {
        if (items.length >= MAX_ITEMS) break;
        if (seen.has(el)) continue;
        if (!isVisible(el)) continue;

        const tagName = el.tagName.toLowerCase();
        const role = el.getAttribute('role');
        const className = (el.className && typeof el.className === 'string') ? el.className.toLowerCase() : "";
        const style = window.getComputedStyle(el);
        const isClickableStyle = style.cursor === 'pointer';

        // =================================================================
        // 🔥 НОВОЕ: RICH TEXT INPUTS (Telegram, WhatsApp, CMS)
        // =================================================================
        const isContentEditable = el.getAttribute('contenteditable') === 'true' || el.isContentEditable;
        const isTextboxRole = role === 'textbox';
        // Ловим спан с плейсхолдером, если он кликабелен (специфика Телеграма)
        const isPlaceholderText = className.includes('placeholder-text') || className.includes('placeholder');

        if (isContentEditable || isTextboxRole || (isPlaceholderText && isClickableStyle)) {
            // Пропускаем, если родитель уже был добавлен как инпут (чтобы не дублировать)
            if (el.parentElement && seen.has(el.parentElement)) continue;

            seen.add(el);
            const id = idCounter++;
            el.setAttribute('data-agent-id', String(id));

            // Пытаемся найти текст плейсхолдера
            let t = el.innerText || el.getAttribute('aria-label') || el.getAttribute('placeholder') || "";
            
            // Если текст пустой, ищем .placeholder внутри (для contenteditable контейнеров)
            if (!t.trim()) {
                const innerPlaceholder = el.querySelector('.placeholder-text, [class*="placeholder"]');
                if (innerPlaceholder) t = innerPlaceholder.innerText;
            }
            
            // Если это сам плейсхолдер (как в твоем примере), берем его текст
            if (isPlaceholderText && !t) t = el.innerText;

            t = t.replace(/[\n\r]+/g, " ").trim().substring(0, 50);
            
            // 🏷️ ВАЖНО: Помечаем как [INPUT], чтобы агент знал, что сюда можно писать
            items.push({ id, tag: 'input', text: "[INPUT] " + (t || "Message Input"), interactive: true });
            continue;
        }

        // =================================================================
        // 1. INPUTS & TEXTAREAS (Стандартные)
        // =================================================================
        if (tagName === 'input' || tagName === 'textarea') {
            seen.add(el);
            const id = idCounter++;
            el.setAttribute('data-agent-id', String(id));
            
            if (el.type === 'checkbox' || el.type === 'radio') {
                let label = "";
                if (el.labels && el.labels.length > 0) label = el.labels[0].innerText;
                const state = el.checked ? ' (V)' : ' ( )';
                items.push({ id, tag: 'checkbox', text: "[SELECT] " + (label || "Checkbox") + state, interactive: true });
            } else if (el.type === 'submit' || el.type === 'button') {
                items.push({ id, tag: 'button', text: "[ACTION] " + (el.value || "Button"), interactive: true });
            } else {
                let t = el.placeholder || el.value || "";
                items.push({ id, tag: 'input', text: "[INPUT] " + (t || "Text Field"), interactive: true });
            }
            continue;
        }

        // =================================================================
        // 2. КАСТОМНЫЕ ЧЕКБОКСЫ
        // =================================================================
        const isLikelyCheckbox = className.includes('checkbox') || role === 'checkbox' || role === 'radio';
        if (isLikelyCheckbox && !el.querySelector('input')) {
            seen.add(el);
            const id = idCounter++;
            el.setAttribute('data-agent-id', String(id));
            const isSelected = className.includes('active') || className.includes('checked') || el.getAttribute('aria-checked') === 'true';
            const state = isSelected ? ' [V]' : ' [ ]';
            let t = (el.innerText || "").replace(/[\n\r]+/g, " ").trim().substring(0, 50);
            items.push({ id, tag: 'custom-checkbox', text: "[SELECT] " + (t || "Option") + state, interactive: true });
            continue;
        }

        // =================================================================
        // 3. ССЫЛКИ
        // =================================================================
        if (tagName === 'a') {
            const href = el.getAttribute('href');
            // Разрешаем ссылки без href, если они кликабельны (SPA навигация)
            if (!href && !el.getAttribute('onclick') && !role && !isClickableStyle) continue;
            
            seen.add(el);
            const id = idCounter++;
            el.setAttribute('data-agent-id', String(id));
            
            let t = el.innerText || el.getAttribute('aria-label') || el.getAttribute('title') || "";
            if (!t) {
                 const img = el.querySelector('img');
                 if (img) t = img.alt || "Image Link";
            }
            t = t.replace(/[\n\r]+/g, " ").trim().substring(0, 50);
            items.push({ id, tag: 'link', text: "[NAVIGATE] " + (t || "Link"), interactive: true });
            continue;
        }

        // =================================================================
        // 4. КНОПКИ
        // =================================================================
        if (tagName === 'button' || role === 'button') {
            seen.add(el);
            const id = idCounter++;
            el.setAttribute('data-agent-id', String(id));
            let t = (el.innerText || el.getAttribute('aria-label') || "Button").replace(/[\n\r]+/g, " ").trim().substring(0, 50);
            items.push({ id, tag: 'button', text: "[ACTION] " + t, interactive: true });
            continue;
        }

        // =================================================================
        // 5. ПРОЧИЕ КЛИКАБЕЛЬНЫЕ (div, span, img)
        // =================================================================
        if ((tagName === 'div' || tagName === 'span' || tagName === 'li' || tagName === 'img' || tagName === 'svg') && isClickableStyle) {
             const rect = el.getBoundingClientRect();
             if (rect.width > 500 && rect.height > 500) continue; 
             
             // Проверка на дублирование с родителем
             let parent = el.parentElement;
             let parentFound = false;
             while(parent && parent !== document.body) {
                if (seen.has(parent)) { parentFound = true; break; }
                parent = parent.parentElement;
             }
             if (parentFound) continue;

             seen.add(el);
             const id = idCounter++;
             el.setAttribute('data-agent-id', String(id));

             let t = el.innerText || el.getAttribute('alt') || "";
             t = t.replace(/[\n\r]+/g, " ").trim().substring(0, 40);
             items.push({ id, tag: 'clickable', text: "[CLICK] " + (t || "Item"), interactive: true });
        }
    }

    return JSON.stringify(items);
}`
