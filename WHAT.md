ms-videos - Microservice de Processamento de Vídeos

ms-videos é um microservice event-driven que automatiza o processamento de vídeos para streaming adaptativo. Desenvolvido em Go, ele consome mensagens de uma fila RabbitMQ contendo URLs de vídeos e os processa automaticamente.

🎯 O que faz:

1. Recebe solicitações via fila RabbitMQ com ID, URL e nome do vídeo
2. Baixa o vídeo da URL fornecida
3. Converte para múltiplas resoluções (1080p, 720p, 480p, 360p)
4. Fragmenta em HLS (HTTP Live Streaming) com segmentos .ts e playlists .m3u8
5. Armazena no MinIO/S3 organizados por resolução

🏗️ Arquitetura:
📁 Resultado:
Cada vídeo processado gera uma estrutura organizada:
🚀 Ideal para:
•  Plataformas de streaming
•  CDNs de vídeo
•  Sistemas de vídeo sob demanda
•  Processamento batch de conteúdo audiovisual

Tecnologias: Go, FFmpeg, RabbitMQ, MinIO/S3, Docker, HLS