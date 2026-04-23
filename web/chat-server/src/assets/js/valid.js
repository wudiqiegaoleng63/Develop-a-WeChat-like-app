export function checkTelephoneValid(telephone) {
    const regex = /^1[3456789]\d{9}$/;
    return regex.test(telephone);
};

export function checkEmailValid(email) {
    const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return regex.test(email);
};

