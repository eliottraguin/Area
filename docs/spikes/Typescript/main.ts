import express, { type Request, type Response } from 'express';
const app = express();

app.get('/', (req: Request, res: Response) => {
  res.send('Hello, World!');
});

app.get('/ping', (req: Request, res: Response) => {
  res.send('{"status":"ok"}');
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});