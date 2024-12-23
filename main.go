// Para rodar o programa, execute o seguinte comando:
// go run main.go
// O programa irá exibir um menu com várias opções para interagir com o sistema de arquivos FURGfs2, os dados dos integrantes do grupo estão dentro de um arquivo já presente no sistema de arquivos ao qual pode ser copiado para o sistema real.

package main
// O pacote main implementa uma aplicação de sistema de arquivos chamada FURGfs2.
// Esta aplicação permite aos usuários interagir com um sistema de arquivos, realizando diversas operações,
// como copiar arquivos, remover arquivos, renomear arquivos, listar arquivos e gerenciar diretórios.
// O sistema de arquivos é armazenado em um arquivo binário e consiste em um cabeçalho, uma tabela de alocação de arquivos (FAT)
// e um diretório raiz. 
// O cabeçalho contém informações sobre o sistema de arquivos, como tamanho total, tamanho dos blocos, espaço livre
// e endereços da FAT e do diretório raiz.
// A FAT é usada para controlar o status de alocação de cada bloco no sistema de arquivos.
// O diretório raiz armazena informações sobre arquivos e diretórios no sistema de arquivos.
// A estrutura FURGFileSystem representa o estado do sistema de arquivos e fornece métodos para operá-lo.

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unsafe"
)

// main é a função principal que inicia a aplicação do sistema de arquivos FURGfs2.
// Ele verifica se um arquivo de sistema de arquivos existente está presente e carrega-o, ou cria um novo sistema de arquivos.
// Em seguida, ele inicia a operação do sistema de arquivos, permitindo que o usuário interaja com ele.
func main() {
	// fmt.Printf("O tamanho de FATEntry e %d bytes \n", unsafe.Sizeof(FATEntry{})) -> 12 bytes, compilador adiciona 3 bytes apos o campo USED para alinha ao tamanho com os outros campos -> Facilita a busca e acesso em memoria
	fileName := "furg.fs2"
	if _, err := os.Stat(fileName); err == nil {
		fmt.Println("Arquivo do sistema de arquivos encontrado. Carregando...")
		fs, err := loadFileSystem(fileName)
		if err != nil {
			fmt.Println("Erro ao carregar o sistema de arquivos:", err)
			return
		}
		fs.operateFileSystem()
	} else {
		fmt.Println("Nenhum sistema de arquivos existente encontrado. Criando um novo...")
		fsSize := getFileSystemSize()
		if fsSize == 0 {
			return
		}
		var blockSize uint32 = 4096
		fs, err := createFileSystem(blockSize, fsSize)
		if err != nil {
			fmt.Println("Erro ao criar o sistema de arquivos:", err)
			return
		}
		fs.operateFileSystem()
	}
}

// loadFileSystem carrega um sistema de arquivos existente de um arquivo binário e retorna uma instância de FURGFileSystem.
// Ele lê o cabeçalho, a FAT e o diretório raiz do arquivo e os armazena na estrutura FURGFileSystem que foram serializados.
// Se ocorrer um erro ao abrir ou ler o arquivo, ele retorna um erro.
func loadFileSystem(fileName string) (*FURGFileSystem, error) {
	f, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o arquivo: %v", err)
	}

	// Ler o cabeçalho
	var header Header
	err = binary.Read(f, binary.LittleEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o cabeçalho: %v", err)
	}

	// Calcular tamanhos
	fatEntrySize := uint32(unsafe.Sizeof(FATEntry{}))
	fatSize := calculateFATSize(header.TotalSize-header.DataStart, header.BlockSize, fatEntrySize)
	entriesNumber := (header.DataStart - header.RootDirStart) / uint32(unsafe.Sizeof(FileEntry{}))

	// Ler a FAT
	fat := make([]FATEntry, fatSize/fatEntrySize)
	for i := range fat {
		err = binary.Read(f, binary.LittleEndian, &fat[i])
		if err != nil {
			return nil, fmt.Errorf("erro ao ler a FAT: %v", err)
		}
	}

	// Ler o diretório raiz
	rootDir := make([]FileEntry, entriesNumber)
	for i := range rootDir {
		err = binary.Read(f, binary.LittleEndian, &rootDir[i])
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("erro ao ler o diretório raiz: %v", err)
		}
	}

	fs := FURGFileSystem{
		Header:      header,
		FAT:         fat,
		RootDir:     rootDir,
		FilePointer: f,
	}

	fmt.Println("Sistema de arquivos carregado com sucesso.")
	return &fs, nil
}

// saveFileSystemState salva o estado atual do sistema de arquivos no arquivo binário.
// Ele escreve o cabeçalho, a FAT e o diretório raiz no arquivo, serializando-os.
// Se ocorrer um erro ao reposicionar o ponteiro do arquivo ou escrever os dados, ele retorna um erro.
func (fs *FURGFileSystem) saveFileSystemState() error {
	// Resetar o arquivo para escrever do início
	_, err := fs.FilePointer.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("erro ao reposicionar ponteiro no arquivo: %v", err)
	}

	// Salvar o cabeçalho
	err = binary.Write(fs.FilePointer, binary.LittleEndian, fs.Header)
	if err != nil {
		return fmt.Errorf("erro ao salvar cabeçalho: %v", err)
	}

	// Salvar a FAT
	for _, entry := range fs.FAT {
		err = binary.Write(fs.FilePointer, binary.LittleEndian, entry)
		if err != nil {
			return fmt.Errorf("erro ao salvar FAT: %v", err)
		}
	}

	// Salvar o diretório raiz
	for _, entry := range fs.RootDir {
		err = binary.Write(fs.FilePointer, binary.LittleEndian, entry)
		if err != nil {
			return fmt.Errorf("erro ao salvar diretório raiz: %v", err)
		}
	}

	return nil
}

// getFileSystemSize exibe um menu para o usuário escolher o tamanho do sistema de arquivos.
func getFileSystemSize() uint32 {
	var size uint32
	running := true
	for running {
		fmt.Println("Escolha sua opção:")
		fmt.Println("1. 10MB")
		fmt.Println("2. 100MB")
		fmt.Println("3. 800MB")
		fmt.Println("4. Sair.")
		consoleScanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Resposta: ")
		consoleScanner.Scan()
		inputStr := consoleScanner.Text()
		option, e := strconv.Atoi(inputStr)
		if e != nil {
			fmt.Printf("Entrada inválida: '%s'. Por favor, insira um número entre 1 e 4.\n", inputStr)
			continue
		}
		switch option {
		case 1:
			size = 10 * 1024 * 1024
		case 2:
			size = 100 * 1024 * 1024
		case 3:
			size = 800 * 1024 * 1024
		case 4:
			running = false
			continue
		default:
			fmt.Println("Opção inválida. Escolha um número entre 1 e 4.")
		}
		return size
	}
	return 0
}

type Header struct {
	TotalSize            uint32
	BlockSize            uint32
	FreeSpace            uint32
	FATEntrypointAddress uint32
	RootDirStart         uint32
	DataStart            uint32
}

type FATEntry struct {
	BlockID     uint32 // 4 bytes de 0 a 2**32 - 1
	NextBlockID uint32 // 4 bytes
	Used        bool   // 1 byte
}

type FileEntry struct {
	Name         [32]byte
	Path         [128]byte
	Size         uint32
	FirstBlockID uint32
	Protected    bool
	IsDirectory  bool
}
type FURGFileSystem struct {
	Header      Header
	FAT         []FATEntry
	RootDir     []FileEntry
	FilePointer *os.File
}

func calculateFATSize(FileSystemSize uint32, BlockSize uint32, FATEntrySize uint32) uint32 {
	totalBlocks := FileSystemSize / BlockSize
	fatSize := totalBlocks * FATEntrySize
	return fatSize
}

func calculateRootDirSize(entriesNumber uint32) uint32 {
	rootDirSize := uint32(entriesNumber) * uint32(unsafe.Sizeof(FileEntry{}))
	return rootDirSize
}

func calculateHeaderSize() uint32 {
	HeaderSize := uint32(unsafe.Sizeof(Header{}))
	return HeaderSize
}

// createFileSystem cria um novo sistema de arquivos com o tamanho total especificado e o tamanho do bloco.
// Ele cria um arquivo binário para armazenar o sistema de arquivos e escreve o cabeçalho inicial no arquivo.
// Em seguida, ele calcula o tamanho da FAT e do diretório raiz com base no tamanho total e no número de entradas.
func createFileSystem(BlockSize uint32, TotalSize uint32) (*FURGFileSystem, error) {
	f, err := os.OpenFile("furg.fs2", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Erro ao abrir/criar o arquivo", err)
		return nil, nil
	} else {
		fmt.Println("Arquivo do FileSystem criado com sucesso com permissao de escrita e leitura.")
	}

	var entriesNumber uint32 = 100

	rootDirSize := calculateRootDirSize(entriesNumber)
	headerSize := calculateHeaderSize()
	fatEntrySize := uint32(unsafe.Sizeof(FATEntry{}))
	FATSize := calculateFATSize(TotalSize-headerSize-rootDirSize, BlockSize, fatEntrySize)

	header := Header{
		TotalSize:            TotalSize,
		BlockSize:            BlockSize,
		FreeSpace:            TotalSize - headerSize - FATSize - rootDirSize,
		FATEntrypointAddress: headerSize,
		RootDirStart:         headerSize + FATSize,
		DataStart:            headerSize + FATSize + rootDirSize,
	}

	err = binary.Write(f, binary.LittleEndian, header)
	if err != nil {
		fmt.Println("Escrita do arquivo em binario falhou.", err)
	}
	fileSystem := FURGFileSystem{
		Header:      header,
		FAT:         make([]FATEntry, FATSize/fatEntrySize),
		RootDir:     make([]FileEntry, entriesNumber),
		FilePointer: f,
	}

	return &fileSystem, nil // Retornar pontiero pois ao inves de duplicar a memoria, apenas retorna o ponteiro de referencia a ele.
}

// operateFileSystem exibe um menu para o usuário escolher uma opção de operação do sistema de arquivos.
// Através desse menu todas as funções do sistema de arquivos são acessadas.
func (fs *FURGFileSystem) operateFileSystem() {
	var option int
	for {
		fmt.Println("\n--- Menu do Sistema de Arquivos FURGfs2 ---")
		fmt.Println("1. Copiar arquivo para o sistema de arquivos")
		fmt.Println("2. Remover arquivo do sistema de arquivos")
		fmt.Println("3. Renomear arquivo armazenado no FURGfs2")
		fmt.Println("4. Listar todos os arquivos armazenados no FURGfs2")
		fmt.Println("5. Listar o espaço livre em relação ao total do FURGfs2")
		fmt.Println("6. Proteger/desproteger arquivo contra escrita/remoção")
		fmt.Println("7. Copiar um arquivo do sistema ficticio para o real")
		fmt.Println("8. Criar diretório")
		fmt.Println("9. Listar diretórios")
		fmt.Println("10. Remover diretório")
		fmt.Println("0. Sair")
		fmt.Print("Escolha uma opção: ")
		fmt.Scanln(&option)

		switch option {
		case 1:
			var externalPath string
			var internalPath string
			var protectionBit int

			fmt.Println("Opção 1: Copiar arquivo para o sistema de arquivos.")

			fmt.Print("Digite o caminho completo do arquivo para copiar: ")
			fmt.Scanln(&externalPath)

			fmt.Print("Digite o caminho completo no FurgFS2 onde o arquivo vai ficar: (digite / para raiz) ")
			fmt.Scanln(&internalPath)

			fmt.Print("Digite o bit de proteção (1 para protegido, 0 para não protegido): ")
			fmt.Scanln(&protectionBit)

			if protectionBit != 0 && protectionBit != 1 {
				fmt.Println("Bit de proteção inválido! Deve ser 1 ou 0.")
				continue
			}
			isProtected := protectionBit == 1

			fs.CopyFileToFileSystem(externalPath, internalPath, isProtected)
		case 2:
			var fileName string
			var path string

			fmt.Println("Opção 2: Remover arquivo do sistema de arquivos.")

			fmt.Print("Digite o nome completo do arquivo(com extensão) para remover: ")
			fmt.Scanln(&fileName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			fmt.Printf("Arquivo '%s' será removido.\n", fileName)
			err := fs.RemoveFileFromFileSystem(fileName, path)
			if err != nil {
				fmt.Println(err)
			}
		case 3:
			var oldName string
			var path string
			var newName string

			fmt.Println("Opção 3: Renomear arquivo armazenado no FURGfs2.")

			fmt.Print("Digite o o nome completo do arquivo(com extensão) a ser renomeado: ")
			fmt.Scanln(&oldName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			fmt.Print("Digite o novo nome do arquivo: ")
			fmt.Scanln(&newName)

			fmt.Printf("Arquivo '%s' será renomeado para '%s'.\n", oldName, newName)
			err := fs.RenameFileFromFileSystem(oldName, path, newName)
			if err != nil {
				fmt.Println(err)
			}
		case 4:
			fmt.Println("Opção 4: Listar todos os arquivos armazenados no FURGfs2.")
			fmt.Println("Listagem de arquivos:")
			fs.ShowAllFilesFromFileSystem()
		case 5:
			fmt.Println("Opção 5: Listar o espaço livre em relação ao total do FURGfs2.")
			fmt.Println("Espaço livre e total:")
			fs.ShowFreeSpaceFromFileSystem()
		case 6:
			var fileName string
			var path string

			fmt.Println("Opção 6: Proteger/desproteger arquivo contra escrita/remoção.")

			fmt.Print("Digite o nome do arquivo a ser protegido/desprotegido: ")
			fmt.Scanln(&fileName)

			fmt.Print("Digite o caminho do arquivo: ")
			fmt.Scanln(&path)

			err := fs.ChangePermission(fileName, path)
			if err != nil {
				fmt.Println(err)
			}
		case 7:
			var fileName string
			var internalPath string
			var externalPath string

			fmt.Print("Digite o nome do arquivo que deseja copiar para o sistema real: ")
			fmt.Scanln(&fileName)

			if fileName == "" {
				fmt.Println("Erro: Nome do arquivo não pode estar vazio.")
				break
			}

			fmt.Print("Digite o caminho do arquivo no FURGfs2: ")
			fmt.Scanln(&internalPath)
			if internalPath == "" {
				fmt.Println("Erro: Caminho do arquivo não pode estar vazio.")
				break
			}

			fmt.Print("Digite o caminho completo onde deseja salvar o arquivo(lembrar de colocar a extensao caso queira abrir o arquivo): ")
			fmt.Scanln(&externalPath)
			if externalPath == "" {
				fmt.Println("Erro: Caminho de destino não pode estar vazio.")
				break
			}

			err := fs.CopyFileFromFileSystem(fileName, internalPath, externalPath)
			if err != nil {
				fmt.Printf("Erro ao copiar o arquivo: %v\n", err)
			} else {
				fmt.Printf("Arquivo '%s' copiado com sucesso para '%s'.\n", fileName, externalPath)
			}
		case 8:
			fmt.Println("Opção 8: Criar diretório.")
			fmt.Print("Digite o nome do diretório a ser criado(Não pode conter /): ")
			var name string
			fmt.Scanln(&name)
			var path string
			fmt.Print("Digite o caminho do diretório pai(Exemplo: /, ou /teste):")
			fmt.Scanln(&path)
			err := fs.CreateDirectory(name, path)

			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("Diretório '%s' criado com sucesso no caminho '%s'.\n", name, path)
			}
		case 9:
			fmt.Println("Opção 9: Listar diretórios.")
			fs.Tree()

		case 10:
			var name string
			var path string

			fmt.Println("Opção 10: Remover diretório.")

			fmt.Print("Digite o nome do diretório a ser removido: ")
			fmt.Scanln(&name)

			fmt.Print("Digite o caminho do diretório pai: ")
			fmt.Scanln(&path)

			err := fs.DeleteDirectory(name, path)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("Diretório '%s' removido com sucesso no caminho '%s'.\n", name, path)
			}
		case 0:
			fmt.Println("Encerrando o sistema de arquivos...")
			err := fs.saveFileSystemState()
			if err != nil {
				fmt.Println("Erro ao salvar o estado do sistema de arquivos:", err)
			} else {
				fmt.Println("Estado do sistema de arquivos salvo com sucesso.")
			}
			return

		default:
			fmt.Println("Opção inválida. Tente novamente.")
		}
	}
}

func (fs *FURGFileSystem) CheckFileEntryAlreadyExists(name [32]byte, path [128]byte) int {
	fileNameStr := string(name[:])
	pathStr := string(path[:])

	for i, v := range fs.RootDir {
		existingFileName := string(v.Name[:])
		existingPath := string(v.Path[:])

		if existingFileName == fileNameStr && existingPath == pathStr {
			return i
		}
	}

	// Retorna -1 se o arquivo não for encontrado
	return -1
}

func (fs *FURGFileSystem) ProcessFileForFileSystem(path string) (*os.File, [32]byte, string, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao abrir o arquivo: %w", err)
	}

	fileInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao obter informações do arquivo: %w", err)
	}

	fileSize := fileInfo.Size()
	if fileSize > int64(fs.Header.FreeSpace) {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o arquivo é muito grande para o espaço disponível")
	}

	var fileSizeUint32 uint32 = uint32(fileSize)

	fileName := filepath.Base(path)

	if len(fileName) > 32 {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o nome do arquivo excede o limite de 32 bytes")
	}

	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	return f, fileNameArray, fileName, fileSizeUint32, nil
}

func (fs *FURGFileSystem) CopyFileToFileSystem(externalPath string, internalPath string, protected bool) bool {
	f, fileNameArray, fileName, fileSizeUint32, err := fs.ProcessFileForFileSystem(externalPath)

	if err != nil {
		fmt.Println(err)
		return false
	}

	var pathArray [128]byte
	copy(pathArray[:], internalPath)

	if cod := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray); cod != -1 {
		fmt.Println("erro: arquivo com o mesmo nome já existe no diretório pai.")
		return false
	}

	buf := make([]byte, fs.Header.BlockSize)

	var firstBlock, previousBlock uint32
	firstBlockSet := false
	for {
		bytesRead, err := f.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("Erro ao ler o arquivo:", err)
			return false
		}
		if bytesRead == 0 {
			break
		}

		var currentBlockID uint32
		found := false
		for i, v := range fs.FAT {
			if !v.Used {
				currentBlockID = uint32(i)
				tmp := FATEntry{
					BlockID:     currentBlockID,
					NextBlockID: 0,
					Used:        true,
				}
				found = true
				fs.FAT[i] = tmp
				fs.Header.FreeSpace -= fs.Header.BlockSize
				break
			}
		}
		if !found {
			fmt.Println("erro: espaço insuficiente na FAT.")
			return false
		}

		if !firstBlockSet {
			firstBlock = currentBlockID
			firstBlockSet = true
		} else {
			fs.FAT[previousBlock].NextBlockID = currentBlockID
		}
		previousBlock = currentBlockID

		_, err = fs.FilePointer.Seek(int64(fs.Header.DataStart+(currentBlockID*fs.Header.BlockSize)), 0)
		if err != nil {
			fmt.Println("Erro ao mover ponteiro do arquivo:", err)
			return false
		}
		_, err = fs.FilePointer.Write(buf[:bytesRead])
		if err != nil {
			fmt.Println("Erro ao escrever dados no arquivo:", err)
			return false
		}
	}

	for i, entry := range fs.RootDir {
		if entry.Name[0] == 0 {
			fs.RootDir[i] = FileEntry{
				Name:         fileNameArray,
				Path:         pathArray,
				Size:         fileSizeUint32,
				FirstBlockID: firstBlock,
				Protected:    protected,
			}
			break
		}
	}
	fmt.Printf("Arquivo '%s' copiado com sucesso para o sistema de arquivos.\n", fileName)
	defer f.Close()
	return true
}

func (fs *FURGFileSystem) CreateDirectory(name string, path string) error {
	var nameArray [32]byte
	copy(nameArray[:], name)

	if bytes.Contains(nameArray[:], []byte("/")) {
		return fmt.Errorf("erro: O nome do diretório não pode conter '/'")
	}

	if isAllNullBytes(name) {
		return fmt.Errorf("erro: Não existem diretórios com nome vazio")
	}

	// verificar se o path existe
	if i := fs.CheckDirectoryExists(path); i == -1 {
		return fmt.Errorf("erro: O caminho '%s' não existe", path)
	}

	// cria entry file
	var pathArray [128]byte
	copy(pathArray[:], path)

	// verifica se já existe um diretório com o mesmo nome dentro do diretório pai
	if i := fs.CheckFileEntryAlreadyExists(nameArray, pathArray); i != -1 {
		return fmt.Errorf("erro: Já existe um diretório com o nome '%s' no diretório pai", name)
	}

	fileEntry := FileEntry{
		Name:        nameArray,
		Path:        pathArray,
		IsDirectory: true,
	}

	err := fs.AddFileEntry(fileEntry)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FURGFileSystem) DeleteDirectory(name, path string) error {
	var nameArray [32]byte
	copy(nameArray[:], name)

	var pathArray [128]byte
	copy(pathArray[:], path)

	rootDirIndex := fs.CheckDirectoryExists(path)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O caminho '%s' não existe", path)
	}

	var completePath string
	if path == "/" {
		completePath = "/" + name
	} else {
		completePath = path + "/" + name
	}

	for _, v := range fs.RootDir {
		trimmedExistingPath := string(bytes.Trim(v.Path[:], "\x00"))

		if trimmedExistingPath == completePath {
			return fmt.Errorf("erro: O diretório '%s' não está vazio", completePath)
		}
	}

	fs.RootDir[rootDirIndex] = FileEntry{}
	return nil
}

func (fs *FURGFileSystem) AddFileEntry(fileEntry FileEntry) error {
	for i, entry := range fs.RootDir {
		if entry.Name[0] == 0 {
			fs.RootDir[i] = fileEntry
			return nil
		}
	}
	return fmt.Errorf("erro: Não foi possível adicionar a entrada de arquivo ao sistema de arquivos")
}

func (fs *FURGFileSystem) CheckDirectoryExists(path string) int {
	if path == "/" {
		return 0
	}

	var completePath string
	for i, v := range fs.RootDir {
		trimmedExistingName := string(bytes.Trim(v.Name[:], "\x00"))
		trimmedExistingPath := string(bytes.Trim(v.Path[:], "\x00"))

		if trimmedExistingPath == "/" {
			completePath = "/" + trimmedExistingName
		} else {
			completePath = trimmedExistingPath + "/" + trimmedExistingName
		}

		if completePath == path && fs.RootDir[i].IsDirectory {
			return i
		}
	}
	return -1

}

func (fs *FURGFileSystem) Tree() {
	fmt.Println("/")
	// Começa listando os arquivos e diretórios sem pai (root)
	for i := range fs.RootDir {
		entry := &fs.RootDir[i]
		name := string(bytes.Trim(entry.Name[:], "\x00")) // Remove bytes nulos do nome
		path := string(bytes.Trim(entry.Path[:], "\x00")) // Remove bytes nulos do path
		if name != "" {
			if path == "/" {
				fmt.Printf("/%s (Size: %d bytes)\n", name, entry.Size)
			} else {
				fmt.Printf("%s/%s (Size: %d bytes)\n", path, name, entry.Size)
			}
		}

	}
}

func (fs *FURGFileSystem) RemoveFileFromFileSystem(fileName, path string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	if isAllNullBytes(fileName) {
		return fmt.Errorf("erro: Não existem arquivos com nome vazio")
	}

	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo '%s' em '%s' não foi armazenado no sistema de arquivos", path, fileName)
	}

	f := fs.RootDir[rootDirIndex]

	if f.Protected {
		return fmt.Errorf("erro: Arquivo protegido, troque sua proteção para poder remover")
	}

	nextBlockId := f.FirstBlockID
	for nextBlockId != 0 {
		currentFileEntry := fs.FAT[nextBlockId]
		nextBlockId = currentFileEntry.NextBlockID
		fs.FAT[nextBlockId] = FATEntry{}
		fs.Header.FreeSpace += fs.Header.BlockSize
	}

	fs.RootDir[rootDirIndex] = FileEntry{}

	fmt.Printf("O arquivo com nome '%s' em '%s' foi removido no sistema de arquivos.\n", fileName, path)
	return nil
}

func (fs *FURGFileSystem) RenameFileFromFileSystem(oldFileName, path, newFileName string) error {
	var oldFileNameArray [32]byte
	copy(oldFileNameArray[:], oldFileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	rootDirIndex := fs.CheckFileEntryAlreadyExists(oldFileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo com nome '%s' não foi armazenado no sistema de arquivos", oldFileName)
	}

	var newFileNameArray [32]byte
	copy(newFileNameArray[:], newFileName)
	if fs.RootDir[rootDirIndex].Protected {
		return fmt.Errorf("erro: Arquivo protegido, troque sua proteção para poder remover")
	}
	fs.RootDir[rootDirIndex].Name = newFileNameArray

	fmt.Printf("arquivo '%s' renomeado, antes era '%s", newFileName, oldFileName)
	return nil
}

func isAllNullBytes(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			return false
		}
	}
	return true
}

func (fs *FURGFileSystem) ShowAllFilesFromFileSystem() {
	for i, file := range fs.RootDir {
		fileName := string(file.Name[:])
		path := string(file.Path[:])

		if fileName != "" && !isAllNullBytes(fileName) && !file.IsDirectory {
			fmt.Printf("%d. %s - path: %s", i, fileName, path)
			fmt.Printf("  -  %s\n", map[bool]string{true: "protegido", false: "desprotegido"}[file.Protected])
		}
	}
}

func (fs *FURGFileSystem) ShowFreeSpaceFromFileSystem() {
	totalSize := (fs.Header.TotalSize) / (1024 * 1024)
	freeSpace := (fs.Header.FreeSpace) / (1024 * 1024)

	occupiedSpace := totalSize - freeSpace
	percentOccupied := (float64(occupiedSpace) / float64(totalSize)) * 100

	fmt.Printf("Espaço total: %d MB\n", totalSize)
	fmt.Printf("Espaço livre: %d MB\n", freeSpace)
	fmt.Printf("Espaço ocupado: %d MB (%.2f%%)\n", occupiedSpace, percentOccupied)
}

func (fs *FURGFileSystem) ChangePermission(fileName, path string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	var pathArray [128]byte
	copy(pathArray[:], path)

	if isAllNullBytes(fileName) {
		return fmt.Errorf("erro: Não existem arquivos com nome vazio")
	}

	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, pathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo com nome '%s' não foi armazenado no sistema de arquivos", fileName)
	}

	f := &fs.RootDir[rootDirIndex]
	fmt.Printf("Mudando a proteção do arquivo, agora é: '%s'\n", map[bool]string{true: "protegido", false: "desprotegido"}[f.Protected])
	f.Protected = !f.Protected

	if f.Protected {
		fmt.Printf("O arquivo '%s' agora está protegido.\n", fileName)
	} else {
		fmt.Printf("O arquivo '%s' agora está desprotegido.\n", fileName)
	}

	return nil
}

func (fs *FURGFileSystem) CopyFileFromFileSystem(fileName, internalPath, externalPath string) error {
	var fileNameArray [32]byte
	copy(fileNameArray[:], []byte(fileName))

	var internalPathArray [128]byte
	copy(internalPathArray[:], []byte(internalPath))

	// Verificar se o nome do arquivo é vazio
	if isAllNullBytes(fileName) {
		return fmt.Errorf("erro: Não existem arquivos com nome vazio")
	}

	// Localizar o arquivo no diretório raiz
	rootDirIndex := fs.CheckFileEntryAlreadyExists(fileNameArray, internalPathArray)
	if rootDirIndex == -1 {
		return fmt.Errorf("erro: O arquivo com nome '%s' não foi encontrado no sistema de arquivos", fileName)
	}

	fileEntry := fs.RootDir[rootDirIndex]

	destFile, err := os.Create(externalPath)
	if err != nil {
		return fmt.Errorf("erro ao criar o arquivo no sistema real: %v", err)
	}
	defer destFile.Close()

	currentBlockID := fileEntry.FirstBlockID
	for currentBlockID != 0 {
		offset := int64(fs.Header.DataStart + (currentBlockID * fs.Header.BlockSize))
		_, err := fs.FilePointer.Seek(offset, 0)
		if err != nil {
			return fmt.Errorf("erro ao mover ponteiro para bloco %d: %v", currentBlockID, err)
		}

		buf := make([]byte, fs.Header.BlockSize)
		bytesRead, err := fs.FilePointer.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("erro ao ler bloco %d: %v", currentBlockID, err)
		}

		_, err = destFile.Write(buf[:bytesRead])
		if err != nil {
			return fmt.Errorf("erro ao escrever dados no arquivo destino: %v", err)
		}

		currentBlockID = fs.FAT[currentBlockID].NextBlockID
	}

	fmt.Printf("Arquivo '%s' copiado com sucesso para o caminho '%s'.\n", fileName, externalPath)
	return nil
}
