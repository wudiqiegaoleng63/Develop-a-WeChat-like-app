export function generateCurrentDate() {
    const now = new Date();
    const year = now.getFullYear().toString().padStart(4, '0');
    const month = (now.getMonth() + 1).toString().padStart(2, '0'); // Months are zero based
    const day = now.getDate().toString().padStart(2, '0');
    return `${year}${month}${day}`;
};
export function generateRandomNumber(length) {
    let randomNumber = '';
    for (let i = 0; i < length; i++) {
        randomNumber += Math.floor(Math.random() * 10).toString();
    }
    return randomNumber;
};
export function generateString(length) {
    const datePart = generateCurrentDate();
    const randomPart = generateRandomNumber(length);
    return datePart + randomPart;
}