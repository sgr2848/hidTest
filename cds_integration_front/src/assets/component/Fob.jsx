import { useEffect, useState } from "react";

const FobComponent = () => {
  const [uid, setUID] = useState([]);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    let socket;

    if (isConnected) {
      socket = new WebSocket("ws://localhost:8080/ws"); // Replace with your WebSocket server address

      socket.onmessage = (event) => {
        const hexUID = event.data; // The received hexadecimal UID data as a string
        const binaryUID = hexToBinary(hexUID);
        setUID(binaryUID);
      };
    }

    // Clean up the socket connection when the component unmounts or after 10 seconds
    return () => {
      if (socket) {
        socket.close();
      }
    };
  }, [isConnected]);

  // Function to convert the received hexadecimal string to binary data
  const hexToBinary = (hexString) => {
    // Remove any non-hexadecimal characters and convert to bytes
    const hexBytes = hexString.replace(/[^0-9a-fA-F]/g, "");
    // Convert to Uint8Array (binary data)
    const binaryData = new Uint8Array(
      hexBytes.match(/.{1,2}/g).map((byte) => parseInt(byte, 16))
    );
    return binaryData;
  };

  // Function to handle button click
  const handleButtonClick = () => {
    setIsConnected(true);

    // Disconnect after 10 seconds
    setTimeout(() => {
      setIsConnected(false);
    }, 10000);
  };

  // Function to convert the binary UID data to a space-separated hexadecimal string
  const binaryToHexString = (array) => {
    return Array.from(array)
      .map((byte) => byte.toString(16).padStart(2, "0"))
      .join(" ");
  };

  return (
    <div>
      <h2>Smart Card Reader</h2>
      <p>UID: {binaryToHexString(uid)}</p>
      <button onClick={handleButtonClick}>
        {isConnected ? "Disconnect" : "Connect"}
      </button>
    </div>
  );
};

export default FobComponent;
