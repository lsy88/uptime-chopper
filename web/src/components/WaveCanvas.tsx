import React, { useEffect, useRef } from 'react';

interface WaveCanvasProps {
  color: string; // RGB string, e.g., "92, 221, 139"
  height?: number;
}

const WaveCanvas: React.FC<WaveCanvasProps> = ({ color, height = 50 }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animationFrameId: number;
    let step = 0;

    // Configuration matches the article's logic but adapted
    const lines = [
      `rgba(${color}, 0.1)`, 
      `rgba(${color}, 0.2)`, 
      `rgba(${color}, 0.2)`
    ];
    
    // Resize handler
    const resizeCanvas = () => {
      const parent = canvas.parentElement;
      if (parent) {
        canvas.width = parent.offsetWidth;
        canvas.height = height;
      }
    };
    
    window.addEventListener('resize', resizeCanvas);
    resizeCanvas(); // Initial resize

    const loop = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      step += 1; // Speed

      const boHeight = canvas.height / 5; // Wave amplitude (height/5 is from article logic roughly)
      const posHeight = canvas.height / 1.5; // Base height of the wave line

      for (let j = lines.length - 1; j >= 0; j--) {
        const angle = (step + j * 50) * Math.PI / 180; // Phase shift
        const deltaHeight = Math.sin(angle) * boHeight;
        const deltaHeightRight = Math.cos(angle) * boHeight;

        ctx.fillStyle = lines[j];
        ctx.beginPath();
        ctx.moveTo(0, posHeight + deltaHeight);
        
        // Bezier curve for wave
        ctx.bezierCurveTo(
          canvas.width / 2, 
          posHeight + deltaHeight - boHeight, 
          canvas.width / 2, 
          posHeight + deltaHeightRight - boHeight, 
          canvas.width, 
          posHeight + deltaHeightRight
        );

        ctx.lineTo(canvas.width, canvas.height);
        ctx.lineTo(0, canvas.height);
        ctx.lineTo(0, posHeight + deltaHeight);
        ctx.fill();
        ctx.closePath();
      }

      animationFrameId = requestAnimationFrame(loop);
    };

    loop();

    return () => {
      window.removeEventListener('resize', resizeCanvas);
      cancelAnimationFrame(animationFrameId);
    };
  }, [color, height]);

  return <canvas ref={canvasRef} style={{ display: 'block', width: '100%', height: `${height}px` }} />;
};

export default WaveCanvas;
