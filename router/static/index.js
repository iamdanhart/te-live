document.getElementById('tip-link').addEventListener('click', function(e) {
    e.preventDefault();
    if (/Mobi|Android/i.test(navigator.userAgent)) {
        window.location = 'venmo://paycharge?txn=pay&recipients=troublesendlbk&note=Tip%20the%20band';
        setTimeout(() => window.location = 'https://venmo.com/u/troublesendlbk', 500);
    } else {
        window.open('https://venmo.com/u/troublesendlbk', '_blank', 'noopener,noreferrer');
    }
});